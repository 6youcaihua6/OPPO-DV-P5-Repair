//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const version = "0.1BETA"

var errSkipped = errors.New("hvcC already contains VPS/SPS/PPS; no repair required (playback not verified)")

func publishOutput(ctx context.Context, src, dest string, stamp time.Time) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		out.Close()
		if !ok {
			os.Remove(dest)
		}
	}()
	if _, err = io.Copy(out, contextReader{ctx, in}); err != nil {
		return err
	}
	if err = out.Sync(); err != nil {
		return err
	}
	if err = out.Close(); err != nil {
		return err
	}
	if err = os.Chtimes(dest, stamp, stamp); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	ok = true
	return nil
}

func preflight(r Report) error {
	if r.VideoNum != 1 || r.Vid.Codec != "hevc" {
		return errors.New("Repair requires exactly one HEVC video track")
	}
	if r.DV == nil || r.DV.Profile != 5 || r.DV.RPUPresent != 1 || r.DV.BLPresent != 1 || r.DV.ELPresent != 0 {
		return errors.New("Repair requires Dolby Vision Profile 5 with BL/RPU and no EL; use Inspect for other profiles")
	}
	if r.HVCCValid && r.Complete {
		return errSkipped
	}
	return nil
}

type BatchResult struct {
	Source string `json:"source"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// A panic affects one input, never the rest of the queue.
func isolatedProcess(ctx context.Context, src, dest, mode string, tools map[string]string, log func(string), stage func(string, string, int64, func() error) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Internal error: %v", r)
			crashLog("batch file", r)
		}
	}()
	return processOne(ctx, src, dest, mode, tools, log, stage)
}

func (g *GUI) showResults(results []BatchResult) {
	copyResults := append([]BatchResult(nil), results...)
	g.enqueue(func() { g.lastResults = copyResults; g.renderResults(copyResults) })
}

func init() {
	// Missing translations fall back to English, never to an unrelated language.
	for _, m := range languages {
		m["headline"] = strings.ReplaceAll(m["headline"], "v3.0", version)
	}
	languages["en"]["addTitle"] = "① Add MKV files or folders (includes subfolders)"
	languages["zh-CN"]["addTitle"] = "① 添加 MKV 或文件夹（拖入文件夹时包含子文件夹）"
	languages["zh-TW"]["addTitle"] = "① 加入 MKV 或資料夾（包含子資料夾）"
	languages["en"]["addedFiles"] = "Added %d MKV files; queue: %d"
	languages["zh-CN"]["addedFiles"] = "已添加 %d 个 MKV；队列共 %d 个"
	languages["en"]["noNewFiles"] = "No new MKV files found (empty folder, unsupported files or duplicates)."
	languages["zh-CN"]["noNewFiles"] = "没有发现新的 MKV：可能文件夹为空、没有 MKV，或文件已经在列表中。"

	languages["en"]["outputTitle"] = "② Output root (blank = first source folder); each run gets a new folder"
	languages["zh-CN"]["outputTitle"] = "② 输出根目录（留空 = 首个片源目录）；每次自动新建结果文件夹"
	languages["en"]["outputNote"] = "Each run has its own folder. Reports contain Chinese, then English."
	languages["zh-CN"]["outputNote"] = "每次任务单独建文件夹；自检报告中文在上、英文在下。"

	for code, pair := range map[string][2]string{
		"zh-TW": {"② 輸出根目錄（留空 = 首個影片目錄）；每次建立結果資料夾", "每次獨立資料夾；檢查報告上方中文、下方英文。"},
		"ja":    {"② 出力先（空欄 = 最初の動画の場所）；毎回フォルダーを作成", "実行ごとに保存先を作成。レポートは中国語、続いて英語。"},
		"ko":    {"② 출력 위치 (빈칸 = 첫 파일 폴더); 매번 새 폴더 생성", "실행마다 별도 폴더. 보고서: 중국어 다음 영어."},
		"fr":    {"② Dossier racine (vide = premier fichier) ; nouveau dossier par exécution", "Un dossier par exécution. Rapport en chinois, puis en anglais."},
		"de":    {"② Zielordner (leer = erster Quellordner); neuer Unterordner pro Lauf", "Eigener Ordner pro Lauf. Bericht auf Chinesisch, dann Englisch."},
		"it":    {"② Cartella base (vuota = primo file); nuova cartella per ogni esecuzione", "Cartella dedicata. Rapporto in cinese, poi in inglese."},
		"es":    {"② Carpeta base (vacía = primer archivo); nueva carpeta por ejecución", "Una carpeta por ejecución. Informe en chino y después en inglés."},
		"pt":    {"② Pasta base (vazia = primeiro ficheiro); nova pasta por execução", "Pasta por execução. Relatório em chinês, depois em inglês."},
		"ru":    {"② Папка вывода (пусто = папка первого файла); подпапка для каждого запуска", "Каждый запуск в своей папке. Отчёт: китайский, затем английский."},
		"ar":    {"② مجلد الإخراج (فارغ = مجلد أول ملف)؛ مجلد جديد لكل تشغيل", "مجلد مستقل لكل تشغيل. التقرير بالصينية ثم الإنجليزية."},
	} {
		languages[code]["outputTitle"] = pair[0]
		languages[code]["outputNote"] = pair[1]
	}

	languages["en"]["inspectPass"] = "Headers passed (playback unverified)"
	languages["en"]["inspectIssue"] = "Issues found"
	languages["en"]["inspectUnsupported"] = "Outside P5 scope"
	languages["en"]["inspectSummary"] = "Headers passed %d | Issues %d | Outside P5 scope %d | Check failed %d | Cancelled %d | Not started %d"
	languages["zh-CN"]["inspectPass"] = "结构通过（未验证实播）"
	languages["zh-CN"]["inspectIssue"] = "发现问题"
	languages["zh-CN"]["inspectUnsupported"] = "非 P5 修复范围"
	languages["zh-CN"]["inspectSummary"] = "结构通过 %d | 发现问题 %d | 非 P5 范围 %d | 检查失败 %d | 取消 %d | 未开始 %d"
	for code, note := range outputNotes {
		languages[code]["outputNote"] = note
	}

	extra := map[string]map[string]string{
		"en":    {"pending": "Pending", "running": "Running", "success": "Success", "skipped": "Skipped", "failed": "Failed", "cancelledFile": "Cancelled", "notStarted": "Not started", "empty": "No files selected", "needFiles": "Add at least one MKV file.", "missing": "Missing %s. Install FFmpeg (ffmpeg + ffprobe) / full MKVToolNix (mkvmerge + mkvextract + mkvpropedit), or place them in tools beside this EXE. Select the EXE in Tool paths.\r\n%s", "batchSummary": "Success %d | Skipped %d | Failed %d | Cancelled %d | Not started %d", "estimate": "Progress is an estimate by stage/output size; 100% means the queue finished, not that every file passed.", "startup": "Ready. Sequential queue; valid P5 headers are skipped; originals are preserved."},
		"zh-CN": {"pending": "等待", "running": "处理中", "success": "成功", "skipped": "已跳过", "failed": "失败", "cancelledFile": "已取消", "notStarted": "未开始", "empty": "尚未选择文件", "needFiles": "请先添加至少一个 MKV 文件。", "missing": "未找到 %s。请安装 FFmpeg（ffmpeg + ffprobe）或完整 MKVToolNix（mkvmerge + mkvextract + mkvpropedit），也可放入程序旁的 tools 目录，并在工具路径框选择 EXE。\r\n%s", "batchSummary": "成功 %d | 跳过 %d | 失败 %d | 取消 %d | 未开始 %d", "estimate": "进度按阶段和输出大小估算；100% 表示队列结束，不代表全部成功。", "startup": "就绪。按顺序处理；完整 P5 头信息自动跳过；原文件保留。"},
	}
	for code, m := range extra {
		for k, v := range m {
			languages[code][k] = v
		}
	}
}

func validateProcessTools(tools map[string]string, mode string) error {
	names := []string{"ffmpeg"}
	if mode == "repair" {
		names = append(names, "mkvmerge", "mkvextract", "mkvpropedit")
	}
	for _, name := range names {
		p := tools[name]
		if p == "" && name == "mkvpropedit" && tools["mkvmerge"] != "" {
			p = filepath.Join(filepath.Dir(tools["mkvmerge"]), "mkvpropedit.exe")
		}
		resolved, err := resolveTool(p)
		if err != nil && (p == "" || name == "mkvpropedit") {
			resolved, err = resolveTool(findTool(name))
		}
		if err != nil {
			return fmt.Errorf("%s: %w", pair("缺少 "+name+"；请安装 FFmpeg / 完整 MKVToolNix 或指定工具 EXE。", "Missing "+name+". Install FFmpeg / full MKVToolNix, place EXEs under tools beside the app, or select Tool paths."), err)
		}
		tools[name] = resolved
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func registryToolDirs() []string {
	var dirs []string
	for _, root := range []syscall.Handle{syscall.HKEY_CURRENT_USER, syscall.HKEY_LOCAL_MACHINE} {
		for _, item := range [][2]string{
			{`SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\mkvtoolnix-gui.exe`, ""},
			{`SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\ffmpeg.exe`, ""},
			{`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\MKVToolNix`, "InstallLocation"},
		} {
			var key syscall.Handle
			if syscall.RegOpenKeyEx(root, wide(item[0]), 0, syscall.KEY_READ, &key) != nil {
				continue
			}
			var typ uint32
			buf := make([]uint16, 4096)
			size := uint32(len(buf) * 2)
			err := syscall.RegQueryValueEx(key, wide(item[1]), nil, &typ, (*byte)(unsafe.Pointer(&buf[0])), &size)
			syscall.RegCloseKey(key)
			if err == nil && typ == syscall.REG_SZ {
				p := strings.Trim(syscall.UTF16ToString(buf), "\"")
				if strings.EqualFold(filepath.Ext(p), ".exe") {
					p = filepath.Dir(p)
				}
				dirs = append(dirs, p)
			}
		}
	}
	return dirs
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func (g *GUI) renderResults(results []BatchResult) {
	var rows []string
	for i, r := range results {
		rows = append(rows, fmt.Sprintf("%d. [%s] %s", i+1, tr(g.language, r.Status), r.Source))
	}
	setText(g.filesEdit, strings.Join(rows, "\r\n"))
}

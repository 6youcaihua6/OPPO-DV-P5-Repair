//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

type logRecord struct{ Time, Text string }
type message struct {
	Key  string
	Args []string
}

func event(key string, args ...string) string {
	b, _ := json.Marshal(message{key, args})
	return "\x1e" + string(b) + "\x1f"
}
func pair(zh, en string) string { return event("pair", zh, en) }

// Formatting happens only on the UI thread, using the language at display time.
func renderMessage(lang, s string) string {
	var out strings.Builder
	for {
		a := strings.IndexByte(s, 0x1e)
		if a < 0 {
			out.WriteString(s)
			break
		}
		out.WriteString(s[:a])
		s = s[a+1:]
		b := strings.IndexByte(s, 0x1f)
		if b < 0 {
			out.WriteString(s)
			break
		}
		var m message
		if json.Unmarshal([]byte(s[:b]), &m) != nil {
			out.WriteString(s[:b])
		} else {
			args := append([]string(nil), m.Args...)
			for i := range args {
				args[i] = renderMessage(lang, args[i])
			}
			template := ""
			switch {
			case m.Key == "outcome":
				if len(args) == 3 {
					template = "[" + args[0] + "] " + args[1] + " " + args[2]
				}
			case m.Key == "pair":
				if len(args) == 2 {
					if strings.HasPrefix(lang, "zh") {
						template = args[0]
					} else {
						template = args[1]
					}
				}
			case strings.HasPrefix(m.Key, "@"):
				template = tr(lang, strings.TrimPrefix(m.Key, "@"))
				for _, a := range args {
					d, ss := strings.Index(template, "%d"), strings.Index(template, "%s")
					pos := d
					if pos < 0 || (ss >= 0 && ss < pos) {
						pos = ss
					}
					if pos >= 0 {
						template = template[:pos] + a + template[pos+2:]
					}
				}
			default:
				template = liveLogText(lang, m.Key)
				for i, a := range args {
					template = strings.ReplaceAll(template, fmt.Sprintf("{%d}", i), a)
				}
			}
			out.WriteString(template)
		}
		s = s[b+1:]
	}
	return out.String()
}

var logLocales = map[string][]string{
	"media":          {"视频输出：{0}", "影片輸出：{0}", "Video output: {0}", "動画出力: {0}", "동영상 출력: {0}", "Vidéo exportée : {0}", "Videoausgabe: {0}", "Video in uscita: {0}", "Vídeo de salida: {0}", "Vídeo de saída: {0}", "Выходное видео: {0}", "ملف الفيديو: {0}"},
	"headers":        {"源文件缺少完整的 VPS/SPS/PPS 参数集，可能影响 OPPO 播放兼容性。", "來源缺少完整 VPS/SPS/PPS，可能影響 OPPO 播放相容性。", "The source lacks a complete VPS/SPS/PPS set, which may affect OPPO playback compatibility.", "元動画の VPS/SPS/PPS が不完全なため、OPPO の再生互換性に影響する可能性があります。", "원본의 VPS/SPS/PPS가 불완전하여 OPPO 재생 호환성에 영향을 줄 수 있습니다.", "Les VPS/SPS/PPS sont incomplets ; la lecture sur OPPO peut être affectée.", "VPS/SPS/PPS sind unvollständig; die OPPO-Wiedergabe kann beeinträchtigt sein.", "VPS/SPS/PPS incompleti: la riproduzione su OPPO potrebbe risentirne.", "VPS/SPS/PPS incompletos; puede afectar a la reproducción en OPPO.", "VPS/SPS/PPS incompletos; a reprodução no OPPO pode ser afetada.", "Неполные VPS/SPS/PPS могут повлиять на воспроизведение OPPO.", "مجموعة VPS/SPS/PPS غير مكتملة وقد تؤثر على تشغيل OPPO."},
	"extracting":     {"正在无损提取 HEVC……", "正在無損擷取 HEVC……", "Extracting HEVC without re-encoding…", "HEVC を無劣化で抽出中…", "HEVC 무손실 추출 중…", "Extraction HEVC sans réencodage…", "HEVC wird verlustfrei extrahiert…", "Estrazione HEVC senza ricodifica…", "Extrayendo HEVC sin recodificar…", "A extrair HEVC sem recodificar…", "Извлечение HEVC без перекодирования…", "جارٍ استخراج HEVC دون إعادة ترميز…"},
	"muxing":         {"正在重新封装，保留原音轨、字幕和章节……", "正在重新封裝，保留原音軌、字幕及章節……", "Remuxing with the original audio, subtitles and chapters…", "元の音声・字幕・チャプターを保持して再格納中…", "원본 오디오, 자막, 챕터를 유지하며 재패키징 중…", "Remuxage avec audio, sous-titres et chapitres d’origine…", "Remux mit Originalton, Untertiteln und Kapiteln…", "Remux con audio, sottotitoli e capitoli originali…", "Remultiplexando con audio, subtítulos y capítulos originales…", "A remultiplexar com áudio, legendas e capítulos originais…", "Пересборка с исходными аудио, субтитрами и главами…", "جارٍ إعادة التغليف مع الصوت والترجمة والفصول الأصلية…"},
	"output":         {"结果文件夹：{0}", "結果資料夾：{0}", "Output folder: {0}", "結果フォルダー: {0}", "결과 폴더: {0}", "Dossier de sortie : {0}", "Ausgabeordner: {0}", "Cartella di uscita: {0}", "Carpeta de salida: {0}", "Pasta de saída: {0}", "Папка результатов: {0}", "مجلد النتائج: {0}"},
	"videos":         {"视频目录：{0}", "影片目錄：{0}", "Videos folder: {0}", "動画フォルダー: {0}", "동영상 폴더: {0}", "Dossier vidéos : {0}", "Videoordner: {0}", "Cartella video: {0}", "Carpeta de vídeos: {0}", "Pasta de vídeos: {0}", "Папка видео: {0}", "مجلد الفيديو: {0}"},
	"reports":        {"报告目录：{0}", "報告目錄：{0}", "Reports folder: {0}", "レポートフォルダー: {0}", "보고서 폴더: {0}", "Dossier des rapports : {0}", "Berichtsordner: {0}", "Cartella rapporti: {0}", "Carpeta de informes: {0}", "Pasta de relatórios: {0}", "Папка отчётов: {0}", "مجلد التقارير: {0}"},
	"report":         {"自检报告：{0}", "檢查報告：{0}", "Inspection report: {0}", "検査レポート: {0}", "검사 보고서: {0}", "Rapport : {0}", "Prüfbericht: {0}", "Rapporto: {0}", "Informe: {0}", "Relatório: {0}", "Отчёт: {0}", "تقرير الفحص: {0}"},
	"processingFile": {"处理文件：{0}", "處理檔案：{0}", "Processing file: {0}", "処理ファイル: {0}", "처리 파일: {0}", "Traitement du fichier : {0}", "Datei wird verarbeitet: {0}", "Elaborazione file: {0}", "Procesando archivo: {0}", "A processar ficheiro: {0}", "Обработка файла: {0}", "معالجة الملف: {0}"},
	"reset":          {"已取消，进度已归零，可开始新任务。", "已取消，進度已歸零，可開始新任務。", "Cancelled. Progress reset; ready for a new task.", "キャンセルしました。進捗をリセットし、次の処理を開始できます。", "취소되었습니다. 진행률이 초기화되어 새 작업을 시작할 수 있습니다.", "Annulé. Progression réinitialisée ; prêt pour une nouvelle tâche.", "Abgebrochen. Fortschritt zurückgesetzt; bereit für eine neue Aufgabe.", "Annullato. Avanzamento azzerato; pronto per una nuova attività.", "Cancelado. Progreso reiniciado; listo para otra tarea.", "Cancelado. Progresso reposto; pronto para nova tarefa.", "Отменено. Прогресс сброшен; можно начать новую задачу.", "أُلغي. أُعيد ضبط التقدم ويمكن بدء مهمة جديدة."},
}

func liveLogText(lang, key string) string {
	values := logLocales[key]
	if len(values) == 0 {
		return key
	}
	index := 2
	for i, l := range languageList {
		if l.code == lang {
			index = i
			break
		}
	}
	if index >= len(values) || values[index] == "" {
		index = 2
	}
	return values[index]
}

func (g *GUI) setStatusNow(s string) {
	g.statusRaw = s
	setText(g.statusEdit, renderMessage(g.language, s))
}
func (g *GUI) refreshLog() {
	setText(g.logEdit, renderLogRecords(g.language, g.logRecords))
	pSend.Call(g.logEdit, EM_SETSEL, ^uintptr(0), ^uintptr(0))
	pSend.Call(g.logEdit, EM_SCROLLCARET, 0, 0)
}
func renderLogRecords(lang string, records []logRecord) string {
	var b strings.Builder
	for _, r := range records {
		fmt.Fprintf(&b, "[%s] %s\r\n", r.Time, renderMessage(lang, r.Text))
	}
	return b.String()
}
func (g *GUI) resetProgressVisuals() {
	atomic.StoreInt32(&g.progressBlocked, 1)
	atomic.StoreInt32(&g.progress, 0)
	g.resetGlobeVisuals()
	if g.progressHwnd != 0 {
		pInvalidate.Call(g.progressHwnd, 0, 0)
	}
}
func (g *GUI) requestCancel() {
	if g.cancel != nil {
		g.resetProgressVisuals()
		g.cancel()
		g.setStatusNow(event("@cancelling"))
		g.appendLog(event("@cancelling"))
	}
}
func (g *GUI) appendLocalizedLog(s string) {
	stamp := time.Now().Format("15:04:05")
	g.logRecords = append(g.logRecords, logRecord{stamp, s})
	if len(g.logRecords) > 2000 {
		g.logRecords = g.logRecords[len(g.logRecords)-2000:]
		g.refreshLog()
		return
	}
	pSend.Call(g.logEdit, EM_SETSEL, ^uintptr(0), ^uintptr(0))
	pSend.Call(g.logEdit, EM_REPLACESEL, 0, uintptr(unsafe.Pointer(wide("["+stamp+"] "+renderMessage(g.language, s)+"\r\n"))))
	pSend.Call(g.logEdit, EM_SCROLLCARET, 0, 0)
}

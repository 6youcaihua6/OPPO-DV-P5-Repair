package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type reportLanguageKey struct{}

func reportLanguage(ctx context.Context) string {
	if s, ok := ctx.Value(reportLanguageKey{}).(string); ok {
		return s
	}
	return "en"
}

// Technical identifiers and file paths are intentionally not translated.
// All explanatory prose comes from the selected language, with no duplicate section.
var reportWords = map[string]string{
	"zh-CN": "自检报告|时间|原始文件|结果目录|模式|导出文件|源文件信息|输出文件信息|检查项目|通过|发现问题|非 P5 修复范围|仅验证元数据、参数头和所列检查；不证明完整解码、画面色彩、实播或音画同步正确。请抽查开头、中间、结尾。|只读自检|修复并自检|导出 HEVC|视频轨|音轨|字幕轨|章节|封面|时长|分辨率|帧率|时间戳|原文件保持不变",
	"zh-TW": "自檢報告|時間|原始檔案|結果目錄|模式|匯出檔案|來源資訊|輸出資訊|檢查項目|通過|發現問題|非 P5 修復範圍|僅驗證中繼資料、參數標頭與所列檢查；不保證完整解碼、畫面色彩、實際播放或影音同步正確。請抽查開頭、中間、結尾。|唯讀自檢|修復並自檢|匯出 HEVC|視訊軌|音訊軌|字幕軌|章節|封面|時長|解析度|影格率|時間戳記|原始檔案保持不變",
	"en":    "Inspection report|Time|Original file|Result folder|Mode|Exported file|Source metadata|Output metadata|Checks|Passed|Issues found|Outside P5 repair scope|Only metadata, parameter headers and listed checks are verified. Full decoding, color, playback and A/V sync are not certified. Check the beginning, middle and end.|Read-only inspection|Repair and inspect|Export HEVC|Video tracks|Audio tracks|Subtitle tracks|Chapters|Cover images|Duration|Resolution|Frame rate|Timestamps|The original file is unchanged",
	"ja":    "検査レポート|日時|元ファイル|結果フォルダー|モード|出力ファイル|元ファイル情報|出力情報|検査項目|合格|問題あり|P5 修復の対象外|メタデータ、パラメーターヘッダーと記載の項目のみ検証します。完全なデコード、色、実機再生や音声同期は保証しません。冒頭・中間・末尾を確認してください。|読み取り専用検査|修復と検査|HEVC 書き出し|映像トラック|音声トラック|字幕トラック|チャプター|カバー画像|再生時間|解像度|フレームレート|タイムスタンプ|元ファイルは変更されません",
	"ko":    "검사 보고서|시간|원본 파일|결과 폴더|모드|출력 파일|원본 정보|출력 정보|검사 항목|통과|문제 발견|P5 복구 대상 아님|메타데이터, 매개변수 헤더 및 명시된 항목만 검사합니다. 전체 디코딩, 색상, 실제 재생 및 음성 동기화는 보장하지 않습니다. 처음, 중간, 끝을 확인하세요.|읽기 전용 검사|복구 및 검사|HEVC 내보내기|영상 트랙|음성 트랙|자막 트랙|챕터|표지 이미지|재생 시간|해상도|프레임 속도|타임스탬프|원본 파일은 변경되지 않습니다",
	"fr":    "Rapport de contrôle|Date|Fichier original|Dossier des résultats|Mode|Fichier exporté|Métadonnées source|Métadonnées de sortie|Contrôles|Réussi|Problèmes détectés|Hors du périmètre P5|Seuls les métadonnées, les en-têtes et les contrôles indiqués sont vérifiés. Le décodage complet, les couleurs, la lecture et la synchronisation ne sont pas garantis. Vérifiez le début, le milieu et la fin.|Contrôle en lecture seule|Réparation et contrôle|Export HEVC|Pistes vidéo|Pistes audio|Sous-titres|Chapitres|Images de couverture|Durée|Résolution|Fréquence d’images|Horodatages|Le fichier original reste inchangé",
	"de":    "Prüfbericht|Zeit|Originaldatei|Ergebnisordner|Modus|Exportierte Datei|Quelldaten|Ausgabedaten|Prüfungen|Bestanden|Probleme gefunden|Außerhalb des P5-Reparaturumfangs|Nur Metadaten, Parameter-Header und aufgeführte Prüfungen werden verifiziert. Vollständige Dekodierung, Farben, Wiedergabe und A/V-Synchronität sind nicht bestätigt. Anfang, Mitte und Ende prüfen.|Schreibgeschützte Prüfung|Reparieren und prüfen|HEVC exportieren|Videospuren|Audiospuren|Untertitelspuren|Kapitel|Coverbilder|Dauer|Auflösung|Bildrate|Zeitstempel|Die Originaldatei bleibt unverändert",
	"it":    "Rapporto di verifica|Data|File originale|Cartella risultati|Modalità|File esportato|Metadati originali|Metadati di uscita|Verifiche|Superato|Problemi rilevati|Fuori dall’ambito P5|Sono verificati solo metadati, intestazioni e controlli elencati. Decodifica completa, colori, riproduzione e sincronizzazione non sono garantiti. Controllare inizio, metà e fine.|Verifica in sola lettura|Riparazione e verifica|Esporta HEVC|Tracce video|Tracce audio|Tracce sottotitoli|Capitoli|Copertine|Durata|Risoluzione|Frequenza fotogrammi|Timestamp|Il file originale resta invariato",
	"es":    "Informe de inspección|Fecha|Archivo original|Carpeta de resultados|Modo|Archivo exportado|Metadatos originales|Metadatos de salida|Comprobaciones|Superado|Problemas detectados|Fuera del ámbito P5|Solo se verifican metadatos, cabeceras y comprobaciones indicadas. No se garantizan decodificación completa, colores, reproducción ni sincronización. Revise inicio, mitad y final.|Inspección de solo lectura|Reparar e inspeccionar|Exportar HEVC|Pistas de vídeo|Pistas de audio|Pistas de subtítulos|Capítulos|Portadas|Duración|Resolución|Fotogramas por segundo|Marcas de tiempo|El archivo original no se modifica",
	"pt":    "Relatório de inspeção|Data|Ficheiro original|Pasta de resultados|Modo|Ficheiro exportado|Metadados originais|Metadados de saída|Verificações|Aprovado|Problemas encontrados|Fora do âmbito P5|Apenas metadados, cabeçalhos e verificações indicadas são validados. Descodificação completa, cores, reprodução e sincronização não são garantidas. Verifique início, meio e fim.|Inspeção só de leitura|Reparar e inspecionar|Exportar HEVC|Faixas de vídeo|Faixas de áudio|Faixas de legendas|Capítulos|Capas|Duração|Resolução|Taxa de fotogramas|Marcas temporais|O ficheiro original não é alterado",
	"ru":    "Отчёт проверки|Время|Исходный файл|Папка результатов|Режим|Выходной файл|Исходные метаданные|Выходные метаданные|Проверки|Пройдено|Обнаружены проблемы|Вне области ремонта P5|Проверены только метаданные, заголовки и перечисленные параметры. Полное декодирование, цвета, воспроизведение и синхронизация не гарантированы. Проверьте начало, середину и конец.|Проверка без изменений|Ремонт и проверка|Экспорт HEVC|Видеодорожки|Аудиодорожки|Субтитры|Главы|Обложки|Длительность|Разрешение|Частота кадров|Временные метки|Исходный файл не изменён",
	"ar":    "تقرير الفحص|الوقت|الملف الأصلي|مجلد النتائج|الوضع|الملف الناتج|بيانات المصدر|بيانات الإخراج|الفحوصات|اجتاز|تم اكتشاف مشاكل|خارج نطاق إصلاح P5|يتم فحص البيانات الوصفية والترويسات والبنود المذكورة فقط. لا نضمن فك الترميز الكامل أو الألوان أو التشغيل الفعلي أو التزامن. افحص البداية والمنتصف والنهاية.|فحص للقراءة فقط|إصلاح وفحص|تصدير HEVC|مسارات الفيديو|مسارات الصوت|مسارات الترجمة|الفصول|صور الغلاف|المدة|الدقة|معدل الإطارات|الطوابع الزمنية|لم يتم تغيير الملف الأصلي",
}

type reportCheck struct {
	Name string
	OK   bool
}

func localizedCheckName(lang, name string) string {
	words, ok := reportWords[lang]
	if !ok {
		words = reportWords["en"]
	}
	w := strings.Split(words, "|")
	index := map[string]int{"主视频轨数量保持": 16, "封面数量保持": 20, "音轨数未减少": 17, "字幕轨道数未减少": 18, "章节数未减少": 19, "文件时长误差≤1秒": 21, "平均帧率标记一致": 23}
	if i, ok := index[name]; ok {
		suffix := " (Δ=0)"
		if i == 21 {
			suffix = " (Δ ≤ 1 s)"
		}
		if i == 23 {
			suffix = " (Δ < 0.02 fps)"
		}
		return w[i] + suffix
	}
	ids := map[string]string{"视频仍为HEVC": "Codec=HEVC", "视频分辨率保持": w[22] + " (Δ=0)", "输出hvcC有VPS/SPS/PPS": "hvcC: VPS/SPS/PPS", "DV Profile未变化": "DV Profile (Δ=0)", "DV RPU存在标志未变化": "DV RPU (Δ=0)"}
	if s, ok := ids[name]; ok {
		return s
	}
	return name
}

func localizedReport(lang, src, root, mode, output string, before Report, after *Report, checks []reportCheck, technical string) string {
	words, ok := reportWords[lang]
	if !ok {
		words = reportWords["en"]
	}
	w := strings.Split(words, "|")
	modeIndex := 13
	if mode == "repair" {
		modeIndex = 14
	}
	if mode == "export" {
		modeIndex = 15
	}
	lines := []string{"OPPO P5 Repair " + version + " — " + w[0], w[1] + ": " + time.Now().Format("2006-01-02 15:04:05"), w[2] + ": " + src, w[3] + ": " + root, w[4] + ": " + w[modeIndex], w[25], "", w[6] + ":"}
	metadata := func(r Report) []string {
		dv := "—"
		if r.DV != nil {
			dv = fmt.Sprintf("P%d; BL=%d RPU=%d EL=%d", r.DV.Profile, r.DV.BLPresent, r.DV.RPUPresent, r.DV.ELPresent)
		}
		return []string{fmt.Sprintf("Codec=%s; DV=%s", r.Vid.Codec, dv), fmt.Sprintf("hvcC=%d B; valid=%t; VPS/SPS/PPS=%d/%d/%d", r.ExtradataSize, r.HVCCValid, r.VPS, r.SPS, r.PPS), fmt.Sprintf("%s=%d; %s=%d; %s=%d; %s=%d; %s=%d", w[16], r.VideoNum, w[17], r.AudioNum, w[18], r.SubNum, w[19], r.Chapters, w[20], r.AttachedPicNum), fmt.Sprintf("%s=%.3f s; %s=%dx%d; %s=%s", w[21], r.Duration, w[22], r.Vid.Width, r.Vid.Height, w[23], r.Vid.AvgFPS)}
	}
	lines = append(lines, metadata(before)...)
	if before.HVCCError != "" {
		lines = append(lines, "hvcC: "+before.HVCCError)
	}
	if output != "" {
		lines = append(lines, "", w[5]+": "+output)
	}
	if after != nil {
		lines = append(lines, "", w[7]+":")
		lines = append(lines, metadata(*after)...)
	}
	status := 9
	if mode == "inspect" {
		finding := inspectReport(before)
		if finding.Status == "inspectUnsupported" {
			status = 11
		} else if finding.Status == "inspectIssue" {
			status = 10
		}
		checks = []reportCheck{{"HEVC / video_tracks=1", before.VideoNum == 1 && before.Vid.Codec == "hevc"}, {"DV Profile=5", before.DV != nil && before.DV.Profile == 5}, {"hvcC", before.HVCCValid}, {"VPS/SPS/PPS", before.Complete && before.VPS > 0 && before.SPS > 0 && before.PPS > 0}, {w[23], before.RateNum > 0 && before.RateDen > 0}}
		if before.DV != nil {
			checks = append(checks, reportCheck{"BL=1 / RPU=1 / EL=0", before.DV.BLPresent == 1 && before.DV.RPUPresent == 1 && before.DV.ELPresent == 0})
		}
	}
	lines = append(lines, "", w[8]+":")
	for _, c := range checks {
		mark := w[9]
		if !c.OK {
			mark = w[10]
			if status != 11 {
				status = 10
			}
		}
		lines = append(lines, mark+" — "+c.Name)
	}
	// Export alone verifies extraction, not the P5 structure.
	if mode == "export" {
		lines = append(lines, w[15]+": "+output)
	} else {
		lines = append(lines, "", w[0]+": "+w[status])
	}
	if technical != "" {
		lines = append(lines, "", technical)
	}
	lines = append(lines, "", w[12])
	return strings.Join(lines, "\n") + "\n"
}

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportLanguageSnapshot(t *testing.T) {
	language := "ja"
	ctx := context.WithValue(context.Background(), reportLanguageKey{}, language)
	language = "fr"
	if reportLanguage(ctx) != "ja" {
		t.Fatal("language snapshot changed", language)
	}
}

func TestInspectionReportFilesUseRequestedLanguage(t *testing.T) {
	for code := range languages {
		t.Run(code, func(t *testing.T) {
			dir := t.TempDir()
			r := inspectFixture()
			r.Complete = false
			r.PPS = 0
			_ = writeInspection("source.mkv", dir, dir, r, func(string) {}, code)
			files, err := filepath.Glob(filepath.Join(dir, "*.txt"))
			if err != nil || len(files) != 1 {
				t.Fatal(files, err)
			}
			data, err := os.ReadFile(files[0])
			if err != nil {
				t.Fatal(err)
			}
			words := strings.Split(reportWords[code], "|")
			if !strings.Contains(string(data), words[10]) {
				t.Fatal("missing issue conclusion", string(data))
			}
			if strings.Count(string(data), "OPPO P5 Repair ") != 1 {
				t.Fatal("duplicate sections")
			}
		})
	}
}

func TestReportsAllLanguages(t *testing.T) {
	for code := range languages {
		words, ok := reportWords[code]
		if !ok {
			t.Errorf("missing %s", code)
			continue
		}
		w := strings.Split(words, "|")
		if len(w) != 26 {
			t.Fatalf("%s: %d fields", code, len(w))
		}
		for _, mode := range []string{"inspect", "repair", "export"} {
			r := inspectFixture()
			r.Complete = false
			r.PPS = 0
			text := localizedReport(code, "source.mkv", "root", mode, "output.mkv", r, &r, []reportCheck{{"PTS", false}}, "")
			if !strings.Contains(text, w[0]) || !strings.Contains(text, w[12]) {
				t.Errorf("%s missing localized text", code)
			}
			if strings.Contains(text, "English log") || strings.Contains(text, "中文日志") {
				t.Errorf("bilingual sections: %s", code)
			}
			if code != "en" && strings.Contains(text, "Issues found") {
				t.Errorf("English finding leaked into %s", code)
			}
			if mode != "export" && !strings.Contains(text, w[10]) {
				t.Errorf("failed check hidden: %s", code)
			}
		}
	}
}

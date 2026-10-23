//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOutputFolderNaming(t *testing.T) {
	root := t.TempDir()
	one := []string{filepath.Join(root, "电影 日本語.mkv")}
	if got := runOutputName(one); got != "电影 日本語" {
		t.Fatal(got)
	}
	season := []string{filepath.Join(root, "Series.Name.S02E01.2160p.mkv"), filepath.Join(root, "Series.Name.S02E02.2160p.mkv")}
	if got := runOutputName(season); got != "Series.Name.S02" {
		t.Fatal(got)
	}
	season = []string{filepath.Join(root, "Season folder", "episode1.mkv"), filepath.Join(root, "Season folder", "episode2.mkv")}
	if got := runOutputName(season); got != "Season folder" {
		t.Fatal(got)
	}
	mixed := []string{filepath.Join(root, "a", "Film A.mkv"), filepath.Join(root, "b", "Film B.mkv")}
	if got := runOutputName(mixed); got != "Film A_Batch" {
		t.Fatal(got)
	}
	now := time.Date(2026, 10, 23, 18, 0, 0, 0, time.Local)
	a, e := createRunOutput(root, one, now)
	if e != nil {
		t.Fatal(e)
	}
	b, e := createRunOutput(root, one, now)
	if e != nil || a == b {
		t.Fatal(a, b, e)
	}
	if filepath.Dir(a) != root || !strings.Contains(filepath.Base(a), "_DV_Fix_20261023_180000") {
		t.Fatal(a)
	}
	c, e := createRunOutput("", one, now)
	if e != nil || filepath.Dir(c) != root {
		t.Fatal(c, e)
	}
	if _, e := createRunOutput(root, nil, now); e == nil {
		t.Fatal("empty input accepted")
	}
}

func TestBilingualReport(t *testing.T) {
	r := validReport()
	r.Vid.Width = 1920
	r.Vid.Height = 1080
	r.AudioNum = 2
	zh, en := reportStart("source.mkv", "output", r)
	text := bilingualReport(zh, en)
	a, b := strings.Index(text, "===== 中文日志 ====="), strings.Index(text, "===== English log =====")
	if a < 0 || b <= a || !strings.Contains(text[b:], "audio=2") {
		t.Fatal(text)
	}
	for _, r := range text[b:] {
		if r >= 0x4e00 && r <= 0x9fff {
			t.Fatal("Chinese diagnostic in English section")
		}
	}
	for _, label := range []string{"视频仍为HEVC", "视频分辨率保持", "输出hvcC有VPS/SPS/PPS", "音轨数未减少", "字幕轨道数未减少", "章节数未减少", "DV Profile未变化", "DV RPU存在标志未变化", "文件时长误差≤1秒", "平均帧率标记一致"} {
		if checkEnglish[label] == "" {
			t.Fatal(label)
		}
	}
	path := filepath.Join(t.TempDir(), "source_DV_Fix_Report.txt")
	if e := txtFile(path, text); e != nil {
		t.Fatal(e)
	}
	bytes, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(bytes), "\xef\xbb\xbf") {
		t.Fatal("missing UTF-8 BOM")
	}
}

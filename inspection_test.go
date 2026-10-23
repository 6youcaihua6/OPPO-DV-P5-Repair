//go:build windows

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func inspectFixture() Report {
	r := validReport()
	r.VPS = 1
	r.SPS = 1
	r.PPS = 1
	r.RateNum = 24000
	r.RateDen = 1001
	return r
}

func TestInspectionFindings(t *testing.T) {
	cases := []struct {
		name, status, reason string
		change               func(*Report)
	}{
		{"complete", "inspectPass", "未验证", func(*Report) {}},
		{"missing sets", "inspectIssue", "VPS/SPS/PPS", func(r *Report) { r.Complete = false; r.SPS = 0 }},
		{"invalid hvcC", "inspectIssue", "hvcC", func(r *Report) { r.HVCCValid = false }},
		{"bad flags", "inspectIssue", "配置标志", func(r *Report) { r.DV.BLPresent = 0 }},
		{"unknown rate", "inspectIssue", "帧率", func(r *Report) { r.RateNum = 0 }},
		{"non-DV", "inspectUnsupported", "未检测到", func(r *Report) { r.DV = nil }},
		{"P8", "inspectUnsupported", "P8", func(r *Report) { r.DV.Profile = 8 }},
		{"multiple videos", "inspectUnsupported", "单个", func(r *Report) { r.VideoNum = 2 }},
		{"AVC", "inspectUnsupported", "HEVC", func(r *Report) { r.Vid.Codec = "h264" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := inspectFixture()
			c.change(&r)
			f := inspectReport(r)
			if f.Status != c.status || !strings.Contains(f.ZH, c.reason) {
				t.Fatal(f)
			}
		})
	}
}

func TestInspectionReportAndSubfolders(t *testing.T) {
	root := t.TempDir()
	videos, reports, e := outputSubdirs(root)
	if e != nil {
		t.Fatal(e)
	}
	if filepath.Base(videos) != "Videos" || filepath.Base(reports) != "Reports" {
		t.Fatal(videos, reports)
	}
	r := inspectFixture()
	r.Complete = false
	r.PPS = 0
	var logs []string
	e = writeInspection("source.mkv", root, reports, r, func(s string) { logs = append(logs, s) })
	var f *inspectionResult
	if !errors.As(e, &f) || f.Status != "inspectIssue" {
		t.Fatal(e)
	}
	files, _ := filepath.Glob(filepath.Join(reports, "*.txt"))
	if len(files) != 1 {
		t.Fatal(files)
	}
	b, _ := os.ReadFile(files[0])
	text := string(b)
	if strings.Contains(text, "发现问题") || !strings.Contains(text, "Issues found") || !strings.Contains(strings.Join(logs, "\n"), "发现问题") {
		t.Fatal(text, logs)
	}
	if strings.Contains(text, "中文日志") || strings.Contains(text, "English log") {
		t.Fatal(text)
	}
	media, _ := os.ReadDir(videos)
	if len(media) != 0 {
		t.Fatal("inspect wrote video")
	}
	filesAtRoot, _ := os.ReadDir(root)
	if len(filesAtRoot) != 2 {
		t.Fatal("unexpected root output", filesAtRoot)
	}
}

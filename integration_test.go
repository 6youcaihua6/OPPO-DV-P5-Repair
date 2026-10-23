//go:build windows

package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Explicit opt-in: synthetic HEVC only; this is not a Dolby Vision playback test.
func TestRealTools(t *testing.T) {
	if os.Getenv("P5_INTEGRATION") != "1" {
		t.Skip("set P5_INTEGRATION=1 and tool paths for real-tool smoke tests")
	}
	tools := map[string]string{"ffmpeg": os.Getenv("P5_FFMPEG"), "ffprobe": os.Getenv("P5_FFPROBE")}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	d := t.TempDir()
	src := filepath.Join(d, "测试 日本語 sample.mkv")
	if out, e := command(ctx, tools["ffmpeg"], "-hide_banner", "-nostdin", "-v", "error", "-f", "lavfi", "-i", "testsrc2=size=128x72:rate=24000/1001", "-t", "1", "-c:v", "libx265", "-x265-params", "log-level=error:pools=1", "-y", src); e != nil {
		t.Fatal(out, e)
	}
	resultDir, e := createRunOutput(d, []string{src}, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	original, _ := os.ReadFile(src)
	hash := sha256.Sum256(original)
	stage := func(_ string, _ string, _ int64, fn func() error) error { return fn() }
	log := func(s string) { t.Log(renderMessage("en", s)) }
	// Failure does not prevent subsequent files from being processed.
	if e := isolatedProcess(ctx, filepath.Join(d, "missing.mkv"), resultDir, "inspect", tools, log, stage); e == nil {
		t.Fatal("missing accepted")
	}
	corrupt := filepath.Join(d, "corrupt.mkv")
	if e := os.WriteFile(corrupt, []byte("not a Matroska file"), 0600); e != nil {
		t.Fatal(e)
	}
	err := isolatedProcess(ctx, corrupt, resultDir, "inspect", tools, log, stage)
	var readFinding *inspectionResult
	if err == nil || errors.As(err, &readFinding) {
		t.Fatal("unreadable media must be a check failure", err)
	}
	if e := isolatedProcess(ctx, src, resultDir, "inspect", tools, log, stage); e == nil {
		t.Fatal("inspection cannot report generic success")
	} else {
		var finding *inspectionResult
		if !errors.As(e, &finding) || finding.Status != "inspectUnsupported" {
			t.Fatal(e)
		}
	}
	if e := isolatedProcess(ctx, src, resultDir, "export", tools, log, stage); e != nil {
		t.Fatal(e)
	}
	for _, pattern := range []string{"*_DV_Fix_Report.txt", "*_DV_Fix_Report_2.txt"} {
		reports, _ := filepath.Glob(filepath.Join(resultDir, "Reports", pattern))
		if len(reports) != 1 {
			t.Fatal(pattern, reports)
		}
		data, e := os.ReadFile(reports[0])
		if e != nil {
			t.Fatal(e)
		}
		text := string(data)
		if strings.Contains(text, "===== 中文日志 =====") || strings.Contains(text, "===== English log =====") || !strings.Contains(text, "The original file is unchanged") || !strings.Contains(text, "Inspection report") {
			t.Fatal(text)
		}
	}
	jsonFiles, _ := filepath.Glob(filepath.Join(resultDir, "*.json"))
	if len(jsonFiles) != 0 {
		t.Fatal("unexpected JSON", jsonFiles)
	}
	if e := isolatedProcess(ctx, src, resultDir, "repair", tools, log, stage); e == nil || !strings.Contains(e.Error(), "Profile 5") {
		t.Fatal("non-DV repair should fail before other tools", e)
	}
	cancelled, stop := context.WithCancel(ctx)
	stop()
	if e := isolatedProcess(cancelled, src, resultDir, "export", tools, log, stage); e == nil {
		t.Fatal("cancel ignored")
	}
	after, _ := os.ReadFile(src)
	if sha256.Sum256(after) != hash {
		t.Fatal("source modified")
	}
	scratch, _ := filepath.Glob(filepath.Join(resultDir, ".oppo-p5-*"))
	if len(scratch) != 0 {
		t.Fatal("scratch left behind", scratch)
	}
	exported, _ := filepath.Glob(filepath.Join(resultDir, "Videos", "*.hevc"))
	if len(exported) != 1 {
		t.Fatal(exported)
	}
	if !strings.HasSuffix(exported[0], "_DV_Fix_HEVC.hevc") {
		t.Fatal(exported[0])
	}
	r, e := probe(ctx, tools["ffprobe"], exported[0])
	if e != nil || r.Vid.Codec != "hevc" {
		t.Fatal(r, e)
	}

	// Exercise the real timestamp/mux/header-edit toolchain with non-DV HEVC.
	mkv := os.Getenv("P5_MKV_DIR")
	if mkv == "" {
		t.Log("MKVToolNix smoke omitted: P5_MKV_DIR unset")
		return
	}
	ts := filepath.Join(d, "source.timestamps.txt")
	if _, e = command(ctx, filepath.Join(mkv, "mkvextract.exe"), src, "timestamps_v2", "0:"+ts); e != nil {
		t.Fatal(e)
	}
	muxed := filepath.Join(d, "muxed.mkv")
	if _, e = command(ctx, filepath.Join(mkv, "mkvmerge.exe"), "-o", muxed, "--timestamps", "0:"+ts, exported[0]); e != nil {
		t.Fatal(e)
	}
	if _, e = command(ctx, filepath.Join(mkv, "mkvpropedit.exe"), muxed, "--edit", "track:v1", "--set", "default-duration=41708333"); e != nil {
		t.Fatal(e)
	}
	verify := filepath.Join(d, "output.timestamps.txt")
	if _, e = command(ctx, filepath.Join(mkv, "mkvextract.exe"), muxed, "timestamps_v2", "0:"+verify); e != nil {
		t.Fatal(e)
	}
	if n, delta, e := compareVideoTimestamps(ts, verify); e != nil || n == 0 || delta > 1 {
		t.Fatal(n, delta, e)
	}
}

//go:build windows

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func validReport() Report {
	return Report{VideoNum: 1, Vid: Stream{Codec: "hevc"}, DV: &DVRecord{Profile: 5, RPUPresent: 1, BLPresent: 1}, Complete: true, HVCCValid: true}
}
func TestPreflight(t *testing.T) {
	r := validReport()
	if !errors.Is(preflight(r), errSkipped) {
		t.Fatal("must skip complete P5")
	}
	r.Complete = false
	if preflight(r) != nil {
		t.Fatal("incomplete P5 should repair")
	}
	r.DV.Profile = 8
	if preflight(r) == nil {
		t.Fatal("P8 must not repair")
	}
	r = validReport()
	r.DV = nil
	if preflight(r) == nil {
		t.Fatal("non-DV")
	}
	r = validReport()
	r.VideoNum = 2
	if preflight(r) == nil {
		t.Fatal("multiple videos")
	}
	r = validReport()
	r.DV.ELPresent = 1
	if preflight(r) == nil {
		t.Fatal("unexpected EL")
	}
}
func TestHVCCBoundaries(t *testing.T) {
	h := make([]byte, 23)
	h[0] = 1
	h[22] = 3
	for _, typ := range []byte{32, 33, 34} {
		h = append(h, typ, 0, 1, 0, 3, typ<<1, 1, 0)
	}
	v, s, p, ok, _ := parseHVCC(h)
	if !ok || v != 1 || s != 1 || p != 1 {
		t.Fatal(v, s, p, ok)
	}
	for n := 0; n < len(h); n++ {
		_, _, _, ok, _ := parseHVCC(h[:n])
		if ok {
			t.Fatalf("accepted truncation %d", n)
		}
	}
	h[26] = 0
	h[27] = 0
	_, _, _, ok, _ = parseHVCC(h)
	if ok {
		t.Fatal("zero NAL accepted")
	}
}
func TestPublishNoOverwriteUnicodeAndTimestamp(t *testing.T) {
	d := t.TempDir()
	src := filepath.Join(d, "源 日本語.hevc")
	dst := filepath.Join(d, "修复.mkv")
	os.WriteFile(src, []byte("original"), 0600)
	stamp := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if e := publishOutput(context.Background(), src, dst, stamp); e != nil {
		t.Fatal(e)
	}
	if e := publishOutput(context.Background(), src, dst, stamp); e == nil {
		t.Fatal("overwrite allowed")
	}
	b, _ := os.ReadFile(src)
	if string(b) != "original" {
		t.Fatal("source changed")
	}
	st, _ := os.Stat(dst)
	if !st.ModTime().Equal(stamp) {
		t.Fatal(st.ModTime())
	}
}
func TestTimestampComparison(t *testing.T) {
	d := t.TempDir()
	a := filepath.Join(d, "a")
	b := filepath.Join(d, "b")
	os.WriteFile(a, []byte("# timestamp format v2\n0\n41.708\n100.5\n"), 0600)
	os.WriteFile(b, []byte("# timestamp format v2\n0\n42\n101\n"), 0600)
	n, delta, e := compareVideoTimestamps(a, b)
	if e != nil || n != 3 || delta > 1 {
		t.Fatal(n, delta, e)
	}
	os.WriteFile(b, []byte("# timestamp format v2\n0\n"), 0600)
	if _, _, e = compareVideoTimestamps(a, b); e == nil {
		t.Fatal("frame count mismatch accepted")
	}
}
func TestLanguageCoverage(t *testing.T) {
	for _, lang := range languageList {
		for key := range languages["en"] {
			if tr(lang.code, key) == key || tr(lang.code, key) == "" {
				t.Fatalf("%s %s", lang.code, key)
			}
		}
	}
	if tr("unknown", "repair") != tr("en", "repair") {
		t.Fatal("fallback")
	}
	if !strings.Contains(tr("en", "headline"), version) {
		t.Fatal("version")
	}
}
func TestIsolation(t *testing.T) {
	d := t.TempDir()
	t.Setenv("LOCALAPPDATA", d)
	src := filepath.Join(d, "bad.mkv")
	os.WriteFile(src, []byte("bad"), 0600)
	stage := func(string, string, int64, func() error) error { panic("test") }
	if e := isolatedProcess(context.Background(), "missing", d, "repair", nil, func(string) {}, stage); e == nil {
		t.Fatal("missing file accepted")
	}
	if e := isolatedProcess(context.Background(), src, d, "inspect", nil, func(string) { panic("fixture panic") }, stage); e == nil || !strings.Contains(e.Error(), "fixture panic") {
		t.Fatal("panic not isolated", e)
	}

}

func TestCancelledPublishAndMissingTool(t *testing.T) {
	d := t.TempDir()
	src := filepath.Join(d, "input")
	dst := filepath.Join(d, "out")
	os.WriteFile(src, []byte("test"), 0600)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := publishOutput(ctx, src, dst, time.Now()); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if _, e := os.Stat(dst); !os.IsNotExist(e) {
		t.Fatal("partial output retained")
	}
	tools := map[string]string{"ffmpeg": filepath.Join(d, "missing.exe")}
	if e := validateProcessTools(tools, "export"); e == nil || !strings.Contains(e.Error(), "FFmpeg") {
		t.Fatal(e)
	}
}

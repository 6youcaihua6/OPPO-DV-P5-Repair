//go:build windows

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLiveLogLanguageAndHistory(t *testing.T) {
	path := `F:\影片\日本語.mkv`
	records := []logRecord{{"12:34:56", event("extracting")}, {"12:35:00", event("headers")}, {"12:35:01", event("media", path)}}
	zh, en, ja := renderLogRecords("zh-CN", records), renderLogRecords("en", records), renderLogRecords("ja", records)
	if !strings.Contains(zh, "正在无损提取") || !strings.Contains(en, "Extracting HEVC") || !strings.Contains(ja, "抽出中") {
		t.Fatal(zh, en, ja)
	}
	for _, text := range []string{zh, en, ja} {
		if !strings.Contains(text, "[12:34:56]") || !strings.Contains(text, path) || strings.Contains(text, "典型征兆") {
			t.Fatal(text)
		}
	}
	outcome := event("outcome", event("@failed"), "source.mkv", pair("输出为空", "Output is empty"))
	got := renderMessage("en", outcome)
	if !strings.Contains(got, "[Failed]") || !strings.Contains(got, "Output is empty") || strings.Contains(got, "输出为空") {
		t.Fatal(got)
	}
	if !strings.Contains(renderMessage("zh-CN", outcome), "输出为空") {
		t.Fatal("could not re-render same event")
	}
	processing := event("@processing", "2", "5", "source.mkv")
	if got := renderMessage("en", processing); got != "Processing 2/5: source.mkv" {
		t.Fatal(got)
	}
	if got := renderMessage("ja", processing); !strings.Contains(got, "処理中 2/5") {
		t.Fatal(got)
	}
	for _, l := range languageList {
		for key, values := range logLocales {
			if len(values) != len(languageList) || liveLogText(l.code, key) == key {
				t.Fatal(l.code, key)
			}
		}
	}
}

func TestCancelResetsProgressAndAnimation(t *testing.T) {
	g := &GUI{globePhase: 8, globePressed: true}
	g.setProgress(63)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				g.setProgress(88)
			}
		}()
	}
	g.resetProgressVisuals()
	wg.Wait()
	g.setProgress(95) // a late worker update after cancellation must be ignored
	if atomic.LoadInt32(&g.progress) != 0 || g.globePhase != 0 || g.globePressed {
		t.Fatal(g.progress, g.globePhase, g.globePressed)
	}
	atomic.StoreInt32(&g.progressBlocked, 0)
	g.setProgress(12)
	if atomic.LoadInt32(&g.progress) != 12 {
		t.Fatal("next task cannot update progress")
	}
}

func TestCommandCancellationHelper(t *testing.T) {
	if marker := os.Getenv("P5_CANCEL_HELPER"); marker != "" {
		if e := os.WriteFile(marker, []byte("running"), 0600); e != nil {
			t.Fatal(e)
		}
		time.Sleep(30 * time.Second)
	}
}

func TestMidCommandCancellation(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "child-started")
	t.Setenv("P5_CANCEL_HELPER", marker)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	exe, _ := os.Executable()
	done := make(chan error, 1)
	go func() { _, err := command(ctx, exe, "-test.run=^TestCommandCancellationHelper$"); done <- err }()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, e := os.Stat(marker); e == nil {
			break
		}
		select {
		case err := <-done:
			t.Fatal("child did not start", err)
		case <-ctx.Done():
			t.Fatal("child start timeout")
		case <-ticker.C:
		}
	}
	cancel()
	select {
	case err := <-done:
		if err == nil || ctx.Err() != context.Canceled {
			t.Fatal(err, ctx.Err())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("child process did not stop after cancellation")
	}
}

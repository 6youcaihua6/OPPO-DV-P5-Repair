//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCollectFoldersAndDeduplicate(t *testing.T) {
	d := t.TempDir()
	sub := filepath.Join(d, "子目录 日本語")
	os.Mkdir(sub, 0700)
	a := filepath.Join(d, "first.MKV")
	b := filepath.Join(sub, "second.mkv")
	for _, p := range []string{a, b, filepath.Join(d, "ignore.mp4")} {
		os.WriteFile(p, []byte("fixture"), 0600)
	}
	got, issues := collectMKVs([]string{d, a, sub}, nil)
	if len(issues) != 0 || len(got) != 2 {
		t.Fatal(got, issues)
	}
	got, issues = collectMKVs([]string{d}, []string{strings.ToUpper(a), b})
	if len(issues) != 0 || len(got) != 0 {
		t.Fatal("duplicate", got, issues)
	}
	empty := t.TempDir()
	got, issues = collectMKVs([]string{empty, filepath.Join(d, "missing")}, nil)
	if len(got) != 0 || len(issues) != 1 {
		t.Fatal(got, issues)
	}
}

// Run the contender in another process so this checks the OS lock, not Go state.
func TestInstanceMutex(t *testing.T) {
	if name := os.Getenv("P5_TEST_MUTEX"); name != "" {
		h, exists, e := acquireInstance(name)
		if e != nil {
			t.Fatal(e)
		}
		defer syscall.CloseHandle(h)
		if !exists {
			t.Fatal("second process did not see lock")
		}
		return
	}
	name := fmt.Sprintf(`Local\OPPO_P5_Test_%d_%d`, os.Getpid(), time.Now().UnixNano())
	h, exists, e := acquireInstance(name)
	if e != nil || exists {
		t.Fatal(exists, e)
	}
	exe, _ := os.Executable()
	cmd := exec.Command(exe, "-test.run=^TestInstanceMutex$", "-test.v")
	cmd.Env = append(os.Environ(), "P5_TEST_MUTEX="+name)
	out, e := cmd.CombinedOutput()
	if e != nil {
		syscall.CloseHandle(h)
		t.Fatal(string(out), e)
	}
	syscall.CloseHandle(h)
	h, exists, e = acquireInstance(name)
	if e != nil || exists {
		t.Fatal("lock not released", exists, e)
	}
	syscall.CloseHandle(h)
}

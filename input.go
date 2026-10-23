//go:build windows

package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// WalkDir does not follow directory symlinks, avoiding junction/symlink loops.
func collectMKVs(paths, existing []string) (files []string, issues []string) {
	known := map[string]bool{}
	for _, p := range existing {
		a, e := filepath.Abs(p)
		if e == nil {
			known[strings.ToLower(filepath.Clean(a))] = true
		}
	}
	add := func(p string) {
		if !strings.EqualFold(filepath.Ext(p), ".mkv") {
			return
		}
		key := strings.ToLower(filepath.Clean(p))
		if !known[key] {
			known[key] = true
			files = append(files, p)
		}
	}
	for _, p := range paths {
		p = strings.Trim(strings.TrimSpace(p), "\"")
		if p == "" {
			continue
		}
		a, e := filepath.Abs(p)
		if e != nil {
			issues = append(issues, e.Error())
			continue
		}
		st, e := os.Stat(a)
		if e != nil {
			issues = append(issues, e.Error())
			continue
		}
		if !st.IsDir() {
			add(a)
			continue
		}
		e = filepath.WalkDir(a, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				issues = append(issues, err.Error())
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if !d.IsDir() {
				add(path)
			}
			return nil
		})
		if e != nil {
			issues = append(issues, e.Error())
		}
	}
	return
}

func acquireInstance(name string) (syscall.Handle, bool, error) {
	h, _, err := kernel32.NewProc("CreateMutexW").Call(0, 0, uintptr(unsafe.Pointer(wide(name))))
	if h == 0 {
		return 0, false, err
	}
	return syscall.Handle(h), err == syscall.Errno(183), nil
}

func activateExisting() {
	// Stable class name works even if the EXE is moved or renamed.
	h, _, _ := user32.NewProc("FindWindowW").Call(uintptr(unsafe.Pointer(wide("OPPO_DV_REPAIR_NATIVE_GUI_30"))), 0)
	if h != 0 {
		user32.NewProc("ShowWindow").Call(h, 9)
		user32.NewProc("SetForegroundWindow").Call(h)
	}
}

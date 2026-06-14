package utils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WaitFor polls cond until true or timeout.
func (e *Env) WaitFor(desc string, timeout time.Duration, cond func() bool) {
	e.t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	e.t.Fatalf("timed out after %s waiting for: %s", timeout, desc)
}

// WaitForContent waits until the file (relative to root) equals want.
func (e *Env) WaitForContent(rel, want string, timeout time.Duration) {
	e.t.Helper()
	e.WaitFor(fmt.Sprintf("%s == %q", rel, want), timeout, func() bool {
		data, err := os.ReadFile(e.Path(rel))
		return err == nil && string(data) == want
	})
}

// WaitForGone waits until a path (relative to root) no longer exists.
func (e *Env) WaitForGone(rel string, timeout time.Duration) {
	e.t.Helper()
	e.WaitFor(rel+" gone", timeout, func() bool { return !e.Exists(rel) })
}

// AssertContent fails unless the file (relative to root) has the given content.
func (e *Env) AssertContent(rel, want string) {
	e.t.Helper()
	if got := e.Read(rel); got != want {
		e.t.Fatalf("%s: got %q want %q", rel, got, want)
	}
}

// BackupContains reports whether any backed-up file contains the given content.
func (e *Env) BackupContains(content string) bool {
	e.t.Helper()
	found := false
	_ = filepath.WalkDir(filepath.Join(e.ConfigDir, ".backups"),
		func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if data, rerr := os.ReadFile(p); rerr == nil && strings.Contains(string(data), content) {
				found = true
			}
			return nil
		})
	return found
}

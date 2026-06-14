// Package utils is the shared harness for gofilesync's end-to-end tests.
//
// It mirrors the structure of the application code (a small reusable package
// rather than a loose helper file). Every test drives the REAL compiled binary
// as a subprocess against real config files and real filesystem changes, so the
// underlying behaviour is verified the way a user actually runs the tool.
//
// The package is organised as:
//   - utils.go  : binary build + shared types
//   - env.go    : the per-test Env and its filesystem/config helpers
//   - exec.go   : running the binary (one-shot and watch subprocesses)
//   - assert.go : polling waits and assertions
package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

// lockedBuffer is a concurrency-safe buffer for capturing subprocess output.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Options configures a generated config file beyond the basic sync entries.
type Options struct {
	DebounceMs          int
	PreservePermissions bool
	Ignore              []string
}

var (
	buildOnce sync.Once
	binPath   string
	buildErr  error
)

// ensureBinary compiles the gofilesync binary once per test process.
func ensureBinary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		root, err := repoRoot()
		if err != nil {
			buildErr = err
			return
		}
		dir, err := os.MkdirTemp("", "gofilesync-bin-")
		if err != nil {
			buildErr = err
			return
		}
		name := "gofilesync"
		if runtime.GOOS == "windows" {
			name += ".exe" // Windows requires the .exe extension to execute.
		}
		binPath = filepath.Join(dir, name)
		cmd := exec.Command("go", "build", "-o", binPath, "./cmd/gofilesync")
		cmd.Dir = root
		if out, e := cmd.CombinedOutput(); e != nil {
			buildErr = fmt.Errorf("build failed: %v\n%s", e, out)
		}
	})
	if buildErr != nil {
		t.Fatalf("could not build binary: %v", buildErr)
	}
	return binPath
}

// repoRoot walks up from the current working directory to find go.mod.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from working directory")
		}
		dir = parent
	}
}

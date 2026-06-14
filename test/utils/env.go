package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Env is an isolated, per-test scratch environment.
type Env struct {
	t          *testing.T
	bin        string
	root       string
	ConfigDir  string
	ConfigPath string
}

// NewEnv creates an isolated environment rooted in t.TempDir().
func NewEnv(t *testing.T) *Env {
	t.Helper()
	bin := ensureBinary(t)
	root := t.TempDir()
	return &Env{
		t:          t,
		bin:        bin,
		root:       root,
		ConfigDir:  filepath.Join(root, ".gofilesync"),
		ConfigPath: filepath.Join(root, "gofilesync.yml"),
	}
}

// Path joins a relative path onto the test root.
func (e *Env) Path(rel string) string { return filepath.Join(e.root, rel) }

// WriteConfig writes a config.yaml with just a debounce and sync entries. Each
// entry is a {title, source, dest} triple of paths relative to the test root.
func (e *Env) WriteConfig(debounceMs int, entries [][3]string) {
	e.WriteConfigOpts(Options{DebounceMs: debounceMs}, entries)
}

// WriteConfigOpts writes a config with extra options (ignore globs, permission
// preservation, summary threshold) plus the given sync entries.
func (e *Env) WriteConfigOpts(opts Options, entries [][3]string) {
	e.t.Helper()
	var b strings.Builder
	fmt.Fprintf(&b, "configDir: %s\n", e.ConfigDir)
	if opts.DebounceMs > 0 {
		fmt.Fprintf(&b, "debounceInMs: %d\n", opts.DebounceMs)
	}
	if opts.PreservePermissions {
		b.WriteString("preservePermissions: true\n")
	}
	if len(opts.Ignore) > 0 {
		b.WriteString("ignore:\n")
		for _, g := range opts.Ignore {
			fmt.Fprintf(&b, "  - %q\n", g)
		}
	}
	b.WriteString("sync:\n")
	for _, en := range entries {
		fmt.Fprintf(&b, "  - title: %s\n", en[0])
		fmt.Fprintf(&b, "    source: %s\n", e.Path(en[1]))
		fmt.Fprintf(&b, "    destination: %s\n", e.Path(en[2]))
	}
	if err := os.WriteFile(e.ConfigPath, []byte(b.String()), 0644); err != nil {
		e.t.Fatalf("write config: %v", err)
	}
}

// WriteRawConfig writes a verbatim config file (for error-path tests).
func (e *Env) WriteRawConfig(content string) {
	e.t.Helper()
	if err := os.WriteFile(e.ConfigPath, []byte(content), 0644); err != nil {
		e.t.Fatalf("write raw config: %v", err)
	}
}

// Write creates/overwrites a file (relative to root) with content.
func (e *Env) Write(rel, content string) {
	e.t.Helper()
	full := e.Path(rel)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		e.t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		e.t.Fatalf("write %s: %v", rel, err)
	}
}

// Mkdir creates a directory (relative to root).
func (e *Env) Mkdir(rel string, mode os.FileMode) {
	e.t.Helper()
	if err := os.MkdirAll(e.Path(rel), mode); err != nil {
		e.t.Fatalf("mkdir %s: %v", rel, err)
	}
}

// Rm removes a path (relative to root).
func (e *Env) Rm(rel string) {
	e.t.Helper()
	if err := os.RemoveAll(e.Path(rel)); err != nil {
		e.t.Fatalf("rm %s: %v", rel, err)
	}
}

// Read returns the content of a file relative to root.
func (e *Env) Read(rel string) string {
	e.t.Helper()
	data, err := os.ReadFile(e.Path(rel))
	if err != nil {
		e.t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

// Exists reports whether a path (relative to root) exists.
func (e *Env) Exists(rel string) bool {
	_, err := os.Stat(e.Path(rel))
	return err == nil
}

// Chmod changes the mode of a file relative to root.
func (e *Env) Chmod(rel string, mode os.FileMode) {
	e.t.Helper()
	if err := os.Chmod(e.Path(rel), mode); err != nil {
		e.t.Fatalf("chmod %s: %v", rel, err)
	}
}

// Mode returns the permission bits of a file relative to root.
func (e *Env) Mode(rel string) os.FileMode {
	e.t.Helper()
	info, err := os.Stat(e.Path(rel))
	if err != nil {
		e.t.Fatalf("stat %s: %v", rel, err)
	}
	return info.Mode().Perm()
}

// Symlink creates a symlink, skipping the test where unsupported.
func (e *Env) Symlink(target, linkPath string) {
	e.t.Helper()
	if err := os.Symlink(target, linkPath); err != nil {
		e.t.Skipf("symlinks unavailable: %v", err)
	}
}

package features

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/utdanningsdirektoratet/gofilesync/test/utils"
)

// Malformed YAML fails clearly, not with a stack trace.
func TestErrorMalformedYAML(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteRawConfig("sync:\n  - title: x\n    source: [unterminated\n")

	out, err := e.RunExpectFail("sync")
	if err == nil {
		t.Fatalf("malformed YAML should fail")
	}
	if !strings.Contains(out, "invalid YAML") {
		t.Fatalf("expected a clear YAML error, got:\n%s", out)
	}
	if strings.Contains(out, "panic") || strings.Contains(out, "goroutine") {
		t.Fatalf("must not panic:\n%s", out)
	}
}

// A destination that is a directory while the source is a file is reported and
// causes a non-zero exit, but the directory is left intact.
func TestErrorDestIsDirectory(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Config", "src/config.php", "dest/config.php"}})
	e.Write("src/config.php", "data")
	e.Write("dest/config.php/inner.txt", "i am a directory")

	out, err := e.RunExpectFail("sync")
	if err == nil {
		t.Fatalf("expected non-zero exit when a sync entry fails")
	}
	if !strings.Contains(out, "failed to sync") {
		t.Fatalf("expected a clear failure summary, got:\n%s", out)
	}
	if !e.Exists("dest/config.php/inner.txt") {
		t.Fatalf("existing destination directory must be left intact")
	}
}

// A write to a read-only destination is reported and exits non-zero, without
// panicking.
func TestErrorReadOnlyDestination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission semantics")
	}
	if isRoot() {
		t.Skip("root bypasses permission checks")
	}
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Config", "src/c.php", "dest/ro/c.php"}})
	e.Write("src/c.php", "data")
	e.Mkdir("dest/ro", 0755) // create it writable...
	e.Chmod("dest/ro", 0500) // ...then make it read-only
	t.Cleanup(func() { e.Chmod("dest/ro", 0755) })

	out, err := e.RunExpectFail("sync")
	if err == nil {
		t.Fatalf("expected non-zero exit on permission failure")
	}
	if strings.Contains(out, "panic") {
		t.Fatalf("must not panic:\n%s", out)
	}
}

// A successful sync must exit 0 (the error-exit path must not false-positive).
func TestErrorCleanSyncExitsZero(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Config", "src/c.php", "dest/c.php"}})
	e.Write("src/c.php", "ok")

	if _, err := e.RunExpectFail("sync"); err != nil {
		t.Fatalf("a clean sync must exit 0, got %v", err)
	}
}

// An empty config file fails clearly.
func TestErrorEmptyConfig(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteRawConfig("")

	out, err := e.RunExpectFail("sync")
	if err == nil {
		t.Fatalf("empty config should fail")
	}
	if strings.Contains(out, "panic") {
		t.Fatalf("must not panic:\n%s", out)
	}
}

// A YAML file using tabs (invalid in YAML) fails with a clear message.
func TestErrorTabsInYAML(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteRawConfig("sync:\n\t- title: x\n")

	out, err := e.RunExpectFail("sync")
	if err == nil {
		t.Fatalf("tabbed YAML should fail")
	}
	if !strings.Contains(out, "invalid YAML") {
		t.Fatalf("expected invalid YAML message, got:\n%s", out)
	}
}

// A sync entry missing its source or destination is rejected (not silently
// ignored).
func TestErrorEntryMissingSourceOrDest(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteRawConfig("configDir: " + e.ConfigDir + "\n" +
		"sync:\n" +
		"  - title: Broken\n" +
		"    source: " + e.Path("src/x") + "\n") // no destination

	out, err := e.RunExpectFail("sync")
	if err == nil {
		t.Fatalf("entry missing destination should fail")
	}
	if !strings.Contains(out, "missing a source or destination") {
		t.Fatalf("expected a clear validation error, got:\n%s", out)
	}
}

// An unreadable source file is reported and exits non-zero, without panicking.
func TestErrorUnreadableSource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission semantics")
	}
	if isRoot() {
		t.Skip("root bypasses permission checks")
	}
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Secret", "src/secret.php", "dest/secret.php"}})
	e.Write("src/secret.php", "classified")
	e.Chmod("src/secret.php", 0000)
	t.Cleanup(func() { e.Chmod("src/secret.php", 0644) })

	out, err := e.RunExpectFail("sync")
	if err == nil {
		t.Fatalf("unreadable source should exit non-zero")
	}
	if strings.Contains(out, "panic") {
		t.Fatalf("must not panic:\n%s", out)
	}
}

func isRoot() bool {
	return os.Geteuid() == 0
}

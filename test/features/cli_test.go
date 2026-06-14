package features

import (
	"strings"
	"testing"

	"github.com/ajxudir/gofilesync/test/utils"
)

// --help prints usage and exits successfully.
func TestCLIHelpSucceeds(t *testing.T) {
	e := utils.NewEnv(t)
	for _, flag := range []string{"--help", "-h", "help"} {
		out, err := e.RunRaw(flag)
		if err != nil {
			t.Fatalf("%s should exit 0, got %v", flag, err)
		}
		if !strings.Contains(out, "Usage:") || !strings.Contains(out, "Commands:") {
			t.Fatalf("%s output missing usage text:\n%s", flag, out)
		}
	}
}

// --version prints a version and exits successfully.
func TestCLIVersionSucceeds(t *testing.T) {
	e := utils.NewEnv(t)
	for _, flag := range []string{"--version", "-v", "version"} {
		out, err := e.RunRaw(flag)
		if err != nil {
			t.Fatalf("%s should exit 0, got %v", flag, err)
		}
		if !strings.Contains(out, "gofilesync") {
			t.Fatalf("%s output missing version:\n%s", flag, out)
		}
	}
}

// An unknown command fails with guidance.
func TestCLIUnknownCommandFails(t *testing.T) {
	e := utils.NewEnv(t)
	out, err := e.RunRaw(e.ConfigPath, "frobnicate")
	if err == nil {
		t.Fatalf("unknown command should fail")
	}
	if !strings.Contains(out, "unknown command") {
		t.Fatalf("expected a helpful error, got:\n%s", out)
	}
}

// An unknown option fails with guidance.
func TestCLIUnknownOptionFails(t *testing.T) {
	e := utils.NewEnv(t)
	out, err := e.RunRaw(e.ConfigPath, "sync", "--frobnicate")
	if err == nil {
		t.Fatalf("unknown option should fail")
	}
	if !strings.Contains(out, "unknown option") {
		t.Fatalf("expected a helpful error, got:\n%s", out)
	}
}

// A bad --poll value fails with guidance instead of a stack trace.
func TestCLIBadPollValueFails(t *testing.T) {
	e := utils.NewEnv(t)
	out, err := e.RunRaw(e.ConfigPath, "watch", "--poll", "abc")
	if err == nil {
		t.Fatalf("non-numeric --poll should fail")
	}
	if !strings.Contains(out, "--poll") {
		t.Fatalf("expected a poll-specific error, got:\n%s", out)
	}
}

// No arguments shows usage and a non-zero (misuse) exit.
func TestCLINoArgsShowsUsage(t *testing.T) {
	e := utils.NewEnv(t)
	out, err := e.RunRaw()
	if err == nil {
		t.Fatalf("no args should be a usage error")
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("expected usage text, got:\n%s", out)
	}
}

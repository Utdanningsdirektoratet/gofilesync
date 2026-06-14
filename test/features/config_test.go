package features

import (
	"testing"

	"github.com/ajxudir/gofilesync/test/utils"
)

// A config with no sync entries fails cleanly (non-zero exit), not a panic.
func TestConfigRejectsEmptySyncList(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteRawConfig("configDir: " + e.ConfigDir + "\ndebounceInMs: 100\n")

	if _, err := e.RunExpectFail("sync"); err == nil {
		t.Fatalf("expected non-zero exit for a config with no sync entries")
	}
}

// A missing config file fails cleanly.
func TestConfigRejectsMissingFile(t *testing.T) {
	e := utils.NewEnv(t)
	// Deliberately do not write any config.

	if _, err := e.RunExpectFail("sync"); err == nil {
		t.Fatalf("expected non-zero exit for a missing config file")
	}
}

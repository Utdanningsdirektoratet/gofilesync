package features

import (
	"strings"
	"testing"

	"github.com/utdanningsdirektoratet/gofilesync/test/utils"
)

// --quiet suppresses the progress overview but still syncs and prints a single
// summary line — the behaviour pipelines/cron jobs want.
func TestQuietSuppressesProgressKeepsSummary(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Plugins", "src/local", "dest/local"}})
	e.Write("src/local/a/x.php", "x")
	e.Write("src/local/b/y.php", "y")

	out := e.Run("sync", "--quiet")

	e.AssertContent("dest/local/a/x.php", "x")
	e.AssertContent("dest/local/b/y.php", "y")

	if strings.Contains(out, "Syncing ") || strings.Contains(out, "  - ") {
		t.Fatalf("--quiet should suppress progress overview, got:\n%s", out)
	}

	if !strings.Contains(out, "Sync complete") {
		t.Fatalf("--quiet should still print a summary, got:\n%s", out)
	}
}

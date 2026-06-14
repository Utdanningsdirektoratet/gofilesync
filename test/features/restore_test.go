package features

import (
	"strings"
	"testing"

	"github.com/ajxudir/gofilesync/test/utils"
)

// Deleting a source whose destination was overwritten restores the original
// from backup.
func TestRestoreFromBackupOnDelete(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Config", "src/config.php", "dest/config.php"}})
	e.Write("dest/config.php", "ORIGINAL")
	e.Write("src/config.php", "INCOMING")
	e.Run("sync")
	e.AssertContent("dest/config.php", "INCOMING")

	e.Rm("src/config.php")
	e.Run("sync")

	e.AssertContent("dest/config.php", "ORIGINAL")
}

// A deleted file entry must be restored exactly once (regression: it used to be
// restored both inline and again during deletion reconciliation).
func TestRestoreHappensOnce(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Config", "src/config.php", "dest/config.php"}})
	e.Write("dest/config.php", "ORIGINAL")
	e.Write("src/config.php", "INCOMING")
	e.Run("sync")

	e.Rm("src/config.php")
	out := e.Run("sync")

	if !strings.Contains(out, "1 restored") {
		t.Fatalf("expected exactly '1 restored' in summary, got:\n%s", out)
	}
}

// Deleting a source whose destination the tool created (no backup) removes the
// destination.
func TestRestoreRemovesDestWhenNoBackup(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Config", "src/config.php", "dest/config.php"}})
	e.Write("src/config.php", "created by sync")
	e.Run("sync")
	e.AssertContent("dest/config.php", "created by sync")

	e.Rm("src/config.php")
	e.Run("sync")

	if e.Exists("dest/config.php") {
		t.Fatalf("destination should be removed when source gone and no backup")
	}
}

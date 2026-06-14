package features

import (
	"testing"

	"github.com/ajxudir/gofilesync/test/utils"
)

// Overwriting an existing destination preserves the original as a backup.
func TestBackupOnOverwrite(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Config", "src/config.php", "dest/config.php"}})
	e.Write("src/config.php", "new content")
	e.Write("dest/config.php", "ORIGINAL destination content")

	e.Run("sync")

	e.AssertContent("dest/config.php", "new content")
	if !e.BackupContains("ORIGINAL destination content") {
		t.Fatalf("expected original destination content to be backed up")
	}
}

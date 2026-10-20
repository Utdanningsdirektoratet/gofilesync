package features

import (
	"testing"

	"github.com/utdanningsdirektoratet/gofilesync/test/utils"
)

// clear-cache wipes the backup tree.
func TestClearCacheRemovesBackups(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Config", "src/config.php", "dest/config.php"}})
	e.Write("dest/config.php", "ORIGINAL")
	e.Write("src/config.php", "INCOMING")
	e.Run("sync")
	if !e.BackupContains("ORIGINAL") {
		t.Fatalf("precondition: backup of ORIGINAL should exist")
	}

	e.Run("clear-cache")

	if e.BackupContains("ORIGINAL") {
		t.Fatalf("clear-cache should have removed backups")
	}
}

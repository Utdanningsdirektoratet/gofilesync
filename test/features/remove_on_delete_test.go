package features

import (
	"testing"

	"github.com/ajxudir/gofilesync/test/utils"
)

// onSourceDelete: remove deletes the managed destination even when a backup exists.
func TestRemoveOnSourceDeleteWithBackup(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteRawConfig(`configDir: ` + e.ConfigDir + `
debounceInMs: 300
sync:
  - title: Config
    source: ` + e.Path("src/config.php") + `
    destination: ` + e.Path("dest/config.php") + `
    onSourceDelete: remove
`)
	e.Write("dest/config.php", "ORIGINAL")
	e.Write("src/config.php", "INCOMING")
	e.Run("sync")
	e.AssertContent("dest/config.php", "INCOMING")

	e.Rm("src/config.php")
	e.Run("sync")

	if e.Exists("dest/config.php") {
		t.Fatalf("destination should be removed when onSourceDelete is remove, even with backup")
	}
}

// Omitting onSourceDelete keeps the existing restore-from-backup behaviour.
func TestDefaultOnSourceDeleteRestoresFromBackup(t *testing.T) {
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

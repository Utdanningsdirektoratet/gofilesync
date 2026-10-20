package features

import (
	"testing"

	"github.com/utdanningsdirektoratet/gofilesync/test/utils"
)

// --dry-run makes no changes on disk.
func TestDryRunMakesNoChanges(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Config", "src/config.php", "dest/config.php"}})
	e.Write("src/config.php", "hello")

	e.Run("sync", "--dry-run")

	if e.Exists("dest/config.php") {
		t.Fatalf("dry-run must not create the destination")
	}
}

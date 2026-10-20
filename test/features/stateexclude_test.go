package features

import (
	"testing"

	"github.com/utdanningsdirektoratet/gofilesync/test/utils"
)

// The tool's own state dir (configDir) must never be synced, even when it lives
// under a watched source — otherwise writing the log re-triggers a sync forever.
func TestStateDirNotSynced(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteRawConfig("configDir: " + e.Path("src/.gofilesync") + "\n" +
		"sync:\n" +
		"  - title: All\n" +
		"    source: " + e.Path("src") + "\n" +
		"    destination: " + e.Path("dest") + "\n")
	e.Write("src/a.php", "a")

	e.Run("sync")

	e.AssertContent("dest/a.php", "a")
	if e.Exists("dest/.gofilesync") {
		t.Fatalf("the tool's state dir must not be synced into the destination")
	}
}

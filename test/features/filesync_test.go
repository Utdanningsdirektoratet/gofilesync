package features

import (
	"strings"
	"testing"

	"github.com/ajxudir/gofilesync/test/utils"
)

func TestFileSyncCreatesFile(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Config", "src/config.php", "dest/config.php"}})
	e.Write("src/config.php", "hello")

	e.Run("sync")

	e.AssertContent("dest/config.php", "hello")
}

func TestFileSyncUpdatesOnChange(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Config", "src/config.php", "dest/config.php"}})
	e.Write("src/config.php", "v1")
	e.Run("sync")
	e.AssertContent("dest/config.php", "v1")

	e.Write("src/config.php", "v2")
	e.Run("sync")
	e.AssertContent("dest/config.php", "v2")
}

func TestFileSyncReSyncIsNoOp(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Config", "src/config.php", "dest/config.php"}})
	e.Write("src/config.php", "same")

	e.Run("sync")

	out := e.Run("sync")
	e.AssertContent("dest/config.php", "same")
	if !strings.Contains(out, "1 unchanged") {
		t.Fatalf("re-sync of an unchanged file should report unchanged, got:\n%s", out)
	}
	if e.BackupContains("same") {
		t.Fatalf("re-sync should not produce a backup")
	}
}

func TestFileSyncSymlinkResolvesContent(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Linked", "src/link.php", "dest/link.php"}})
	e.Write("src/target.php", "real content")
	e.Symlink(e.Path("src/target.php"), e.Path("src/link.php"))

	e.Run("sync")

	e.AssertContent("dest/link.php", "real content")
}

func TestFileSyncEmptyFile(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Empty", "src/empty.php", "dest/empty.php"}})
	e.Write("src/empty.php", "")

	e.Run("sync")

	e.AssertContent("dest/empty.php", "")
}

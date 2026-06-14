package features

import (
	"testing"

	"github.com/ajxudir/gofilesync/test/utils"
)

func TestDirectorySyncRecursive(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Plugins", "src/local", "dest/local"}})
	e.Write("src/local/a.php", "alpha")
	e.Write("src/local/sub/b.php", "beta")
	e.Write("src/local/sub/deep/c.php", "gamma")

	e.Run("sync")

	e.AssertContent("dest/local/a.php", "alpha")
	e.AssertContent("dest/local/sub/b.php", "beta")
	e.AssertContent("dest/local/sub/deep/c.php", "gamma")
}

func TestDirectorySyncCreatesNewNestedFile(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Plugins", "src/local", "dest/local"}})
	e.Write("src/local/a.php", "alpha")
	e.Run("sync")

	e.Write("src/local/sub/new.php", "newfile")
	e.Run("sync")

	e.AssertContent("dest/local/sub/new.php", "newfile")
	e.AssertContent("dest/local/a.php", "alpha")
}

// A file removed from the source tree is removed at the destination on the next
// sync (deletion propagation for managed files).
func TestDirectorySyncPropagatesDeletion(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Plugins", "src/local", "dest/local"}})
	e.Write("src/local/keep.php", "keep")
	e.Write("src/local/gone.php", "gone")
	e.Run("sync")
	e.AssertContent("dest/local/gone.php", "gone")

	e.Rm("src/local/gone.php")
	e.Run("sync")

	if e.Exists("dest/local/gone.php") {
		t.Fatalf("deleted source file should be removed at destination")
	}
	e.AssertContent("dest/local/keep.php", "keep")
}

// Safety: files that exist only at the destination (never synced by the tool)
// are never deleted by deletion reconciliation.
func TestDirectorySyncLeavesUnmanagedFiles(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Plugins", "src/local", "dest/local"}})
	e.Write("src/local/managed.php", "managed")
	e.Write("dest/local/preexisting.php", "DO NOT DELETE ME")
	e.Run("sync")

	// Remove the only source file and re-sync.
	e.Rm("src/local/managed.php")
	e.Run("sync")

	if !e.Exists("dest/local/preexisting.php") {
		t.Fatalf("unmanaged pre-existing destination file must not be deleted")
	}
	e.AssertContent("dest/local/preexisting.php", "DO NOT DELETE ME")
}

package features

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ajxudir/gofilesync/test/utils"
)

const massN = 400

// writeMassTree writes massN files spread across 20 subdirectories, each with
// the given content, under the relative root dir.
func writeMassTree(e *utils.Env, dir, content string) {
	for i := 0; i < massN; i++ {
		e.Write(fmt.Sprintf("%s/sub%d/file%d.txt", dir, i%20, i), content)
	}
}

// Deleting a whole directory tree while watching must propagate every deletion
// without logging "Could not sync" errors. Sources can vanish between the walk
// and the worker stat/open (a TOCTOU race) — that's a deletion, not a failure,
// and erroring would flood the log and block deletion reconciliation.
func TestMassiveDeleteUnderWatch(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(0, [][3]string{{"Tree", "src/tree", "dest/tree"}})
	writeMassTree(e, "src/tree", "v1")

	w := e.StartPoll(50)
	w.WaitForOutput("Sync complete", 15*time.Second)
	if got := e.CountFiles("dest/tree"); got != massN {
		t.Fatalf("init synced %d files, want %d", got, massN)
	}

	e.Rm("src/tree")

	e.WaitFor("dest/tree fully removed", 15*time.Second, func() bool {
		return e.CountFiles("dest/tree") == 0
	})
	assertNoSyncError(t, w)
}

// Rewriting every file while watching must re-sync them all (mtime changes are
// detected) without errors.
func TestMassiveChangeUnderWatch(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(0, [][3]string{{"Tree", "src/tree", "dest/tree"}})
	writeMassTree(e, "src/tree", "v1")

	w := e.StartPoll(50)
	w.WaitForOutput("Sync complete", 15*time.Second)

	writeMassTree(e, "src/tree", "version-two")
	e.WaitForContent("dest/tree/sub0/file0.txt", "version-two", 15*time.Second)
	e.WaitForContent("dest/tree/sub19/file399.txt", "version-two", 15*time.Second)

	assertNoSyncError(t, w)
}

// Renaming a large subtree (a "move") while watching must remove the old
// destination paths and create the new ones, with no errors.
func TestMassiveMoveUnderWatch(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(0, [][3]string{{"Tree", "src/tree", "dest/tree"}})
	for i := 0; i < massN; i++ {
		e.Write(fmt.Sprintf("src/tree/old/file%d.txt", i), "v1")
	}

	w := e.StartPoll(50)
	w.WaitForOutput("Sync complete", 15*time.Second)
	e.WaitForContent("dest/tree/old/file0.txt", "v1", 15*time.Second)

	e.Rename("src/tree/old", "src/tree/new")

	e.WaitForContent("dest/tree/new/file0.txt", "v1", 15*time.Second)
	e.WaitForGone("dest/tree/old/file0.txt", 15*time.Second)
	assertNoSyncError(t, w)
}

// Deleting a large tree whose destination pre-existed must restore every file
// from backup (not remove it), with no errors. This exercises the "restored"
// path at scale.
func TestMassiveRestoreUnderWatch(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(0, [][3]string{{"Tree", "src/tree", "dest/tree"}})

	writeMassTree(e, "dest/tree", "original")
	writeMassTree(e, "src/tree", "incoming")

	w := e.StartPoll(50)
	w.WaitForOutput("Sync complete", 15*time.Second)
	e.WaitForContent("dest/tree/sub0/file0.txt", "incoming", 15*time.Second)

	e.Rm("src/tree")
	e.WaitForContent("dest/tree/sub0/file0.txt", "original", 15*time.Second)
	e.WaitForContent("dest/tree/sub19/file399.txt", "original", 15*time.Second)

	e.WaitFor("all files restored", 15*time.Second, func() bool {
		return e.CountFiles("dest/tree") == massN
	})
	assertNoSyncError(t, w)
}

func assertNoSyncError(t *testing.T, w *utils.WatchProc) {
	t.Helper()
	if strings.Contains(w.Output(), "Could not sync") {
		t.Fatalf("output must not contain sync errors:\n%s", w.Output())
	}
}

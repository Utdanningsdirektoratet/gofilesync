package features

import (
	"testing"
	"time"

	"github.com/ajxudir/gofilesync/test/utils"
)

func onceConfig(e *utils.Env) {
	e.WriteRawConfig("configDir: " + e.ConfigDir + "\n" +
		"debounceInMs: 80\n" +
		"sync:\n" +
		"  - title: Live\n" +
		"    source: " + e.Path("src/live") + "\n" +
		"    destination: " + e.Path("dest/live") + "\n" +
		"  - title: Once\n" +
		"    source: " + e.Path("src/once") + "\n" +
		"    destination: " + e.Path("dest/once") + "\n" +
		"    once: true\n")
}

// `sync` (init) syncs once entries like any other.
func TestOnceSyncedOnInit(t *testing.T) {
	e := utils.NewEnv(t)
	onceConfig(e)
	e.Write("src/live/a.php", "live1")
	e.Write("src/once/b.php", "once1")

	e.Run("sync")

	e.AssertContent("dest/live/a.php", "live1")
	e.AssertContent("dest/once/b.php", "once1")
}

// In watch mode a once entry is NOT re-synced on change; live entries are.
func TestOnceIgnoredDuringWatch(t *testing.T) {
	e := utils.NewEnv(t)
	onceConfig(e)
	e.Write("src/live/a.php", "live1")
	e.Write("src/once/b.php", "once1")

	e.StartPoll(80)
	e.WaitForContent("dest/once/b.php", "once1", 5*time.Second)
	e.WaitForContent("dest/live/a.php", "live1", 5*time.Second)

	e.Write("src/once/b.php", "once2") // must be ignored
	e.Write("src/live/a.php", "live2") // proves the watcher is processing
	e.WaitForContent("dest/live/a.php", "live2", 5*time.Second)

	if got := e.Read("dest/once/b.php"); got != "once1" {
		t.Fatalf("once entry must not update during watch; got %q", got)
	}
}

// A once entry's destination survives watch even if its source disappears.
func TestOnceDestinationSurvivesWatch(t *testing.T) {
	e := utils.NewEnv(t)
	onceConfig(e)
	e.Write("src/live/a.php", "live1")
	e.Write("src/once/b.php", "once1")

	e.StartPoll(80)
	e.WaitForContent("dest/once/b.php", "once1", 5*time.Second)

	e.Rm("src/once/b.php")
	e.Write("src/live/a.php", "live2") // trigger a watch batch
	e.WaitForContent("dest/live/a.php", "live2", 5*time.Second)

	if !e.Exists("dest/once/b.php") {
		t.Fatalf("once destination must survive watch when its source is removed")
	}
}

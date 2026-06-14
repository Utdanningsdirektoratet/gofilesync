package features

import (
	"testing"
	"time"

	"github.com/ajxudir/gofilesync/test/utils"
)

// Polling is the primary watch mode: filesystem notifications are unreliable on
// mounted volumes (e.g. Docker on macOS), which is the environment this tool is
// built for. These tests use `watch --poll`.

func TestWatchPollingSyncsOnChange(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(100, [][3]string{{"Plugins", "src/local", "dest/local"}})
	e.Write("src/local/version.php", "v1")

	e.StartPoll(100)
	e.WaitForContent("dest/local/version.php", "v1", 5*time.Second)

	e.Write("src/local/version.php", "v2")
	e.WaitForContent("dest/local/version.php", "v2", 5*time.Second)
}

func TestWatchPollingCreatesNewFile(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(100, [][3]string{{"Plugins", "src/local", "dest/local"}})
	e.Write("src/local/seed.php", "seed")

	e.StartPoll(100)
	e.WaitForContent("dest/local/seed.php", "seed", 5*time.Second)

	// A brand-new nested file (in a new subdirectory) is picked up.
	e.Write("src/local/newmod/feature.php", "feature")
	e.WaitForContent("dest/local/newmod/feature.php", "feature", 5*time.Second)
}

func TestWatchPollingRestoresOnDelete(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(100, [][3]string{{"Config", "src/config.php", "dest/config.php"}})
	e.Write("dest/config.php", "ORIGINAL")
	e.Write("src/config.php", "INCOMING")

	e.StartPoll(100)
	e.WaitForContent("dest/config.php", "INCOMING", 5*time.Second)

	e.Rm("src/config.php")
	e.WaitForContent("dest/config.php", "ORIGINAL", 5*time.Second)
}

// Secondary mode: fsnotify-based watching (used where notifications work).
func TestWatchFsnotifySyncsOnChange(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(100, [][3]string{{"Plugins", "src/local", "dest/local"}})
	e.Write("src/local/version.php", "v1")

	e.StartWatch()
	e.WaitForContent("dest/local/version.php", "v1", 5*time.Second)

	e.Write("src/local/version.php", "v2")
	e.WaitForContent("dest/local/version.php", "v2", 5*time.Second)
}

// Regression test: a directory created at runtime is added to the fsnotify
// watcher, so a later edit to a file inside it is still detected.
func TestWatchFsnotifyRuntimeSubdir(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(100, [][3]string{{"Plugins", "src/local", "dest/local"}})
	e.Write("src/local/seed.php", "seed")

	e.StartWatch()
	e.WaitForContent("dest/local/seed.php", "seed", 5*time.Second)

	// New subdirectory + file created live.
	e.Write("src/local/newmod/v.php", "v1")
	e.WaitForContent("dest/local/newmod/v.php", "v1", 5*time.Second)

	// Editing the file in the runtime-created subdir must still propagate.
	e.Write("src/local/newmod/v.php", "v2")
	e.WaitForContent("dest/local/newmod/v.php", "v2", 5*time.Second)
}

// Debounce (fsnotify mode) coalesces a rapid burst of writes into the final
// state.
func TestWatchDebounceCoalescesBurst(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(200, [][3]string{{"Plugins", "src/local", "dest/local"}})
	e.Write("src/local/f.php", "v0")

	e.StartWatch()
	e.WaitForContent("dest/local/f.php", "v0", 5*time.Second)

	for i := 0; i < 20; i++ {
		e.Write("src/local/f.php", "burst")
		time.Sleep(5 * time.Millisecond)
	}
	e.WaitForContent("dest/local/f.php", "burst", 5*time.Second)
}

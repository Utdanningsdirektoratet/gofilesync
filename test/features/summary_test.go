package features

import (
	"testing"
	"time"

	"github.com/ajxudir/gofilesync/test/utils"
)

// A single changed file is reported with its path on one line.
func TestWatchSmallChangeLogsPerFile(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfigOpts(utils.Options{DebounceMs: 100},
		[][3]string{{"Plugins", "src/local", "dest/local"}})
	e.Write("src/local/a.php", "v1")

	w := e.StartPoll(100)
	e.WaitForContent("dest/local/a.php", "v1", 5*time.Second)

	e.Write("src/local/a.php", "v2")
	e.WaitForContent("dest/local/a.php", "v2", 5*time.Second)

	// A single change is one concise line: "▸ <Title>  (changed <path>)".
	w.WaitForOutput("Plugins  (changed a.php)", 5*time.Second)
}

// Many files changed at once collapse to a single count line (no flood).
func TestWatchLargeBatchLogsSummary(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfigOpts(utils.Options{DebounceMs: 150},
		[][3]string{{"Plugins", "src/local", "dest/local"}})
	e.Write("src/local/seed.php", "seed")

	w := e.StartPoll(100)
	e.WaitForContent("dest/local/seed.php", "seed", 5*time.Second)

	// Drop a plugin: many files at once.
	for i := 0; i < 12; i++ {
		e.Write("src/local/newplugin/file"+itoa(i)+".php", "x")
	}
	e.WaitForContent("dest/local/newplugin/file11.php", "x", 5*time.Second)

	// The batch should be summarized in one counts line, not logged per file.
	w.WaitForOutput("12 changed", 5*time.Second)
}

package projects

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ajxudir/gofilesync/test/utils"
)

// Models syncing composer-installed plugins from a vendor tree into the Moodle
// web root: a large initial sync (where progress matters) while excluding VCS
// and dependency noise via ignore patterns.
func TestComposerVendorSyncWithIgnores(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfigOpts(utils.Options{
		DebounceMs: 200,
		Ignore:     []string{".git/**", "node_modules", "*.md", "tests/**"},
	}, [][3]string{
		{"Restful plugin", "vendor/restful", "html/webservice/restful"},
		{"Local greetings", "vendor/greetings", "html/local/greetings"},
	})

	// A realistic vendor layout with plenty of files plus ignorable noise.
	for i := 0; i < 40; i++ {
		e.Write("vendor/restful/classes/c"+strconv.Itoa(i)+".php", "code")
	}
	e.Write("vendor/restful/version.php", "v1")
	e.Write("vendor/restful/README.md", "docs")           // ignored (*.md)
	e.Write("vendor/restful/.git/config", "vcs")          // ignored (.git/**)
	e.Write("vendor/restful/node_modules/x/i.js", "deps") // ignored (node_modules)
	e.Write("vendor/restful/tests/test_x.php", "test")    // ignored (tests/**)
	e.Write("vendor/greetings/version.php", "g1")

	out := e.Run("sync")

	// Init progress is shown for the large sync.
	if !strings.Contains(out, "[100%]") || !strings.Contains(out, "Initialization complete") {
		t.Fatalf("expected init progress + summary, got:\n%s", out)
	}

	// Real plugin files synced.
	e.AssertContent("html/webservice/restful/version.php", "v1")
	e.AssertContent("html/webservice/restful/classes/c0.php", "code")
	e.AssertContent("html/local/greetings/version.php", "g1")

	// Noise excluded.
	for _, junk := range []string{
		"html/webservice/restful/README.md",
		"html/webservice/restful/.git",
		"html/webservice/restful/node_modules",
		"html/webservice/restful/tests",
	} {
		if e.Exists(junk) {
			t.Fatalf("expected %q to be ignored", junk)
		}
	}
}

// After the initial sync, editing a single vendor file in watch mode propagates
// (polling mode, the reliable mode for mounted volumes).
func TestComposerVendorWatchEdit(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfigOpts(utils.Options{DebounceMs: 150, Ignore: []string{".git/**"}},
		[][3]string{{"Restful plugin", "vendor/restful", "html/webservice/restful"}})
	e.Write("vendor/restful/version.php", "v1")

	e.StartPoll(100)
	e.WaitForContent("html/webservice/restful/version.php", "v1", 5*time.Second)

	e.Write("vendor/restful/version.php", "v2")
	e.WaitForContent("html/webservice/restful/version.php", "v2", 5*time.Second)
}

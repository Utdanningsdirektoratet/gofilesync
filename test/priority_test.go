package projects

import (
	"strings"
	"testing"

	"github.com/ajxudir/gofilesync/test/utils"
)

// When multiple entries target the same destination, the lowest-listed entry
// owns each shared file. After the first sync, re-syncing must report no further
// "synced" (no churn) — the owner wrote it once and earlier entries skip it.
func TestPriorityNoChurnOnReSync(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{
		{"Vendor", "vendor/p", "www/p"},
		{"Override", "project/p", "www/p"},
	})
	e.Write("vendor/p/a.php", "va")
	e.Write("vendor/p/shared.php", "vendor shared")
	e.Write("project/p/shared.php", "PROJECT shared")

	e.Run("sync")
	e.AssertContent("www/p/shared.php", "PROJECT shared")

	out := e.Run("sync") // nothing changed
	if strings.Contains(out, "synced") {
		t.Fatalf("expected no churn (0 synced) on re-sync, got:\n%s", out)
	}
}

// With three overlapping entries, the last one wins per shared file, and
// each file is owned (written) by exactly one entry — so re-sync has no churn.
func TestPriorityMultipleEntriesLastWins(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{
		{"Base", "a/p", "www/p"},
		{"Mid", "b/p", "www/p"},
		{"Top", "c/p", "www/p"},
	})
	e.Write("a/p/x.php", "A")
	e.Write("a/p/y.php", "A") // only in Base
	e.Write("b/p/x.php", "B")
	e.Write("c/p/x.php", "C")  // Top wins for x
	e.Write("b/p/z.php", "Bz") // only in Mid

	e.Run("sync")
	e.AssertContent("www/p/x.php", "C")
	e.AssertContent("www/p/y.php", "A")
	e.AssertContent("www/p/z.php", "Bz")

	out := e.Run("sync")
	if strings.Contains(out, "synced") {
		t.Fatalf("expected no churn across 3 entries, got:\n%s", out)
	}
}

// Removing a higher-priority override hands ownership to the next entry down.
func TestPriorityRevertToLowerEntry(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{
		{"Base", "a/p", "www/p"},
		{"Mid", "b/p", "www/p"},
		{"Top", "c/p", "www/p"},
	})
	e.Write("a/p/x.php", "A")
	e.Write("b/p/x.php", "B")
	e.Write("c/p/x.php", "C")

	e.Run("sync")
	e.AssertContent("www/p/x.php", "C")

	e.Rm("c/p/x.php")
	e.Run("sync")
	e.AssertContent("www/p/x.php", "B")

	e.Rm("b/p/x.php")
	e.Run("sync")
	e.AssertContent("www/p/x.php", "A")
}

// Non-overlapping destinations are unaffected (no ownership pre-pass needed).
func TestPriorityDistinctDestinationsUnaffected(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{
		{"One", "src/one", "www/one"},
		{"Two", "src/two", "www/two"},
	})
	e.Write("src/one/f.php", "1")
	e.Write("src/two/f.php", "2")

	e.Run("sync")
	e.AssertContent("www/one/f.php", "1")
	e.AssertContent("www/two/f.php", "2")
}

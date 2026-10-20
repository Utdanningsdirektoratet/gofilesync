package projects

import (
	"strings"
	"testing"

	"github.com/utdanningsdirektoratet/gofilesync/test/utils"
)

// Three entries targeting the same destination: the last (highest priority)
// wins for a shared file, and re-syncing reports no churn (the ownership
// pre-scan makes each shared file written by exactly one entry).
func TestPriorityMultipleEntriesNoChurn(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{
		{"Base", "base/p", "www/p"},
		{"Mid", "mid/p", "www/p"},
		{"Top", "top/p", "www/p"},
	})
	// shared.php exists in all three; each layer also has a unique file.
	e.Write("base/p/shared.php", "base shared")
	e.Write("base/p/base.php", "base only")
	e.Write("mid/p/shared.php", "mid shared")
	e.Write("top/p/shared.php", "TOP shared")
	e.Write("top/p/top.php", "top only")

	e.Run("sync")
	e.AssertContent("www/p/shared.php", "TOP shared") // last entry wins
	e.AssertContent("www/p/base.php", "base only")
	e.AssertContent("www/p/top.php", "top only")

	// Re-sync: everything unchanged — no churn from the overlapping shared file.
	out := e.Run("sync")
	if !strings.Contains(out, "unchanged") || strings.Contains(out, " synced") {
		t.Fatalf("expected an all-unchanged re-sync (no churn), got:\n%s", out)
	}
}

// Two directory entries targeting the same destination: a vendor plugin plus a
// project "overrides" dir that holds only the changed files. The override entry
// is listed second, so its files win — without per-file ignore rules.
func TestOverlappingEntriesOverrideByOrder(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{
		{"Vendor", "vendor/plugin", "www/plugin"},
		{"Override", "project/plugin", "www/plugin"},
	})
	e.Write("vendor/plugin/a.php", "vendor a")
	e.Write("vendor/plugin/shared.php", "vendor shared")
	e.Write("project/plugin/shared.php", "PROJECT override")

	e.Run("sync")

	e.AssertContent("www/plugin/a.php", "vendor a")              // vendor-only
	e.AssertContent("www/plugin/shared.php", "PROJECT override") // override wins

	// Dropping a new override file takes effect with no config change.
	e.Write("project/plugin/a.php", "PROJECT a")
	e.Run("sync")
	e.AssertContent("www/plugin/a.php", "PROJECT a")

	// Removing an override reverts the destination to the vendor version.
	e.Rm("project/plugin/shared.php")
	e.Run("sync")
	e.AssertContent("www/plugin/shared.php", "vendor shared")
}

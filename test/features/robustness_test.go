package features

import (
	"strings"
	"testing"

	"github.com/ajxudir/gofilesync/test/utils"
)

// Binary content (all byte values, including NULs) must round-trip exactly.
func TestRobustBinaryFileIntegrity(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Bin", "src/data.bin", "dest/data.bin"}})
	e.Write("src/data.bin", allBytes())

	e.Run("sync")

	e.AssertContent("dest/data.bin", allBytes())
}

// A multi-megabyte file syncs correctly (exercises streaming copy + hashing).
func TestRobustLargeFileSync(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Big", "src/big.bin", "dest/big.bin"}})
	big := strings.Repeat("0123456789abcdef", 256*1024) // 4 MiB
	e.Write("src/big.bin", big)

	e.Run("sync")

	if got := e.Read("dest/big.bin"); len(got) != len(big) {
		t.Fatalf("large file size mismatch: got %d want %d", len(got), len(big))
	}
	// A second sync should detect it as unchanged (hash compare).
	out := e.Run("sync")
	if !strings.Contains(out, "1 unchanged") {
		t.Fatalf("expected the large file to be skipped as unchanged, got:\n%s", out)
	}
}

// --dry-run over a directory tree must create nothing and write no state.
func TestRobustDryRunDirectoryTree(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Tree", "src/local", "dest/local"}})
	e.Write("src/local/a/x.php", "x")
	e.Write("src/local/b/y.php", "y")

	e.Run("sync", "--dry-run")

	if e.Exists("dest/local") {
		t.Fatalf("--dry-run must not create the destination tree")
	}
	if e.Exists(".gofilesync") {
		t.Fatalf("--dry-run must not write state")
	}
}

// clear-cache --dry-run must not actually remove backups.
func TestRobustClearCacheDryRunKeepsBackups(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Config", "src/c.php", "dest/c.php"}})
	e.Write("dest/c.php", "ORIGINAL")
	e.Write("src/c.php", "INCOMING")
	e.Run("sync") // creates a backup of ORIGINAL

	e.Run("clear-cache", "--dry-run")

	if !e.BackupContains("ORIGINAL") {
		t.Fatalf("clear-cache --dry-run must keep backups")
	}
}

// An ignore pattern that matches everything syncs nothing, gracefully (no
// divide-by-zero in progress, exit 0).
func TestRobustIgnoreEverything(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfigOpts(utils.Options{Ignore: []string{"**"}},
		[][3]string{{"All", "src/local", "dest/local"}})
	e.Write("src/local/a.php", "a")
	e.Write("src/local/sub/b.php", "b")

	out, err := e.RunExpectFail("sync")
	if err != nil {
		t.Fatalf("ignoring everything should still exit 0, got %v\n%s", err, out)
	}
	if e.Exists("dest/local/a.php") || e.Exists("dest/local/sub/b.php") {
		t.Fatalf("nothing should have been synced")
	}
}

// An empty source directory is handled without error.
func TestRobustEmptySourceDirectory(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Empty", "src/empty", "dest/empty"}})
	e.Mkdir("src/empty", 0755)

	if _, err := e.RunExpectFail("sync"); err != nil {
		t.Fatalf("empty source dir should not error, got %v", err)
	}
}

// allBytes returns a string containing every byte value 0-255.
func allBytes() string {
	b := make([]byte, 256)
	for i := range b {
		b[i] = byte(i)
	}
	return string(b)
}

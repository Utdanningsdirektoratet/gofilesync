package features

import (
	"testing"

	"github.com/utdanningsdirektoratet/gofilesync/test/utils"
)

// A global ignore pattern excludes matching files anywhere in the tree.
func TestIgnoreExcludesByBasename(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfigOpts(utils.Options{Ignore: []string{"*.log", "node_modules"}},
		[][3]string{{"App", "src/app", "dest/app"}})
	e.Write("src/app/index.php", "keep")
	e.Write("src/app/debug.log", "drop")
	e.Write("src/app/node_modules/pkg/index.js", "drop")

	e.Run("sync")

	e.AssertContent("dest/app/index.php", "keep")
	if e.Exists("dest/app/debug.log") {
		t.Fatalf("*.log should have been ignored")
	}
	if e.Exists("dest/app/node_modules") {
		t.Fatalf("node_modules should have been ignored")
	}
}

// A "vendor/**" style pattern excludes a whole subtree (composer vendor dir).
func TestIgnoreExcludesSubtree(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfigOpts(utils.Options{Ignore: []string{"vendor/**", ".git/**"}},
		[][3]string{{"App", "src/app", "dest/app"}})
	e.Write("src/app/main.php", "main")
	e.Write("src/app/vendor/autoload.php", "drop")
	e.Write("src/app/vendor/lib/deep/file.php", "drop")
	e.Write("src/app/.git/config", "drop")

	e.Run("sync")

	e.AssertContent("dest/app/main.php", "main")
	if e.Exists("dest/app/vendor") {
		t.Fatalf("vendor/** should have been ignored")
	}
	if e.Exists("dest/app/.git") {
		t.Fatalf(".git/** should have been ignored")
	}
}

// Per-entry ignore patterns apply to just that entry (merged with globals).
func TestIgnorePerEntry(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteRawConfig("configDir: " + e.ConfigDir + "\n" +
		"sync:\n" +
		"  - title: A\n" +
		"    source: " + e.Path("src/a") + "\n" +
		"    destination: " + e.Path("dest/a") + "\n" +
		"    ignore:\n" +
		"      - \"*.tmp\"\n" +
		"  - title: B\n" +
		"    source: " + e.Path("src/b") + "\n" +
		"    destination: " + e.Path("dest/b") + "\n")
	e.Write("src/a/keep.php", "a")
	e.Write("src/a/scratch.tmp", "drop")
	e.Write("src/b/scratch.tmp", "kept-in-b") // not ignored for entry B

	e.Run("sync")

	e.AssertContent("dest/a/keep.php", "a")
	if e.Exists("dest/a/scratch.tmp") {
		t.Fatalf("entry A should ignore *.tmp")
	}
	e.AssertContent("dest/b/scratch.tmp", "kept-in-b")
}

// A "." in a pattern is matched literally, not as a regex wildcard.
func TestIgnoreDotIsLiteral(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfigOpts(utils.Options{Ignore: []string{"a.php"}},
		[][3]string{{"App", "src/app", "dest/app"}})
	e.Write("src/app/a.php", "drop")
	e.Write("src/app/axphp.php", "keep") // must NOT match "a.php"

	e.Run("sync")

	if e.Exists("dest/app/a.php") {
		t.Fatalf("a.php should be ignored")
	}
	e.AssertContent("dest/app/axphp.php", "keep")
}

// A "!" negation re-includes a path otherwise excluded.
func TestIgnoreNegationReincludes(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfigOpts(utils.Options{Ignore: []string{"*.log", "!keep.log"}},
		[][3]string{{"App", "src/app", "dest/app"}})
	e.Write("src/app/a.log", "drop")
	e.Write("src/app/keep.log", "keep")

	e.Run("sync")

	if e.Exists("dest/app/a.log") {
		t.Fatalf("a.log should be ignored")
	}
	e.AssertContent("dest/app/keep.log", "keep")
}

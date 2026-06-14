package features

import (
	"strings"
	"testing"

	"github.com/ajxudir/gofilesync/test/utils"
)

// The initial sync prints one concise line per entry (title + file count) and a
// completion summary — never the directory tree or per-file lines (init runs on
// every dev startup, so it must stay quiet).
func TestInitProgressShowsEntryAndSummary(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Local plugins", "src/local", "dest/local"}})
	e.Write("src/local/greetings/version.php", "v")
	e.Write("src/local/mod/quiz/lib.php", "q")
	e.Write("src/local/theme/boost/config.php", "t")

	out := e.Run("sync")

	if !strings.Contains(out, "Local plugins") || !strings.Contains(out, "(3 files)") {
		t.Fatalf("expected a one-line entry with a file count, got:\n%s", out)
	}
	if !strings.Contains(out, "Sync complete") {
		t.Fatalf("expected a completion summary, got:\n%s", out)
	}

	for _, noise := range []string{"greetings/", "quiz/", "boost/", "version.php"} {
		if strings.Contains(out, noise) {
			t.Fatalf("init should not show directory/file detail, found %q:\n%s", noise, out)
		}
	}
}

// Init stays tiny regardless of how many files/directories exist.
func TestInitProgressIsConcise(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Big", "src/big", "dest/big"}})
	const files = 200
	for i := 0; i < files; i++ {
		e.Write("src/big/dir"+itoa(i%30)+"/file"+itoa(i)+".php", "x")
	}

	out := e.Run("sync")

	if !strings.Contains(out, "Big") || !strings.Contains(out, "(200 files)") {
		t.Fatalf("expected one entry line with a count, got:\n%s", out)
	}
	if lines := strings.Count(out, "\n"); lines > 12 {
		t.Fatalf("init output has %d lines — should be tiny:\n%s", lines, out)
	}
}

// A single-file config entry shows as "(1 file)".
func TestInitProgressShowsSingleFile(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Config", "src/config.php", "dest/config.php"}})
	e.Write("src/config.php", "cfg")

	out := e.Run("sync")

	if !strings.Contains(out, "Config") || !strings.Contains(out, "(1 file)") {
		t.Fatalf("expected the single file shown as '(1 file)', got:\n%s", out)
	}
}

// itoa avoids importing strconv across multiple small test files.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

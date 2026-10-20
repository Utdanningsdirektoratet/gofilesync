package projects

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/utdanningsdirektoratet/gofilesync/test/utils"
)

// Chaos / battle tests: deliberately hostile scenarios meant to find weak links
// and prove the tool degrades gracefully (no hangs, no panics, correct final
// state) under stress.

// A sustained storm of writes across many files while watching must converge.
func TestChaosWriteStorm(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfigOpts(utils.Options{DebounceMs: 120},
		[][3]string{{"Plugins", "src/local", "dest/local"}})
	const files = 30
	for i := 0; i < files; i++ {
		e.Write("src/local/f"+strconv.Itoa(i)+".php", "init")
	}

	e.StartPoll(80)
	e.WaitForContent("dest/local/f0.php", "init", 5*time.Second)

	for round := 0; round < 25; round++ {
		for i := 0; i < files; i++ {
			e.Write("src/local/f"+strconv.Itoa(i)+".php", "r"+strconv.Itoa(round))
		}
		time.Sleep(3 * time.Millisecond)
	}
	for i := 0; i < files; i++ {
		e.WaitForContent("dest/local/f"+strconv.Itoa(i)+".php", "r24", 10*time.Second)
	}
}

// Concurrent goroutines hammering separate entries must all land.
func TestChaosConcurrentWriters(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfigOpts(utils.Options{DebounceMs: 120},
		[][3]string{{"A", "src/a", "dest/a"}, {"B", "src/b", "dest/b"}, {"C", "src/c", "dest/c"}})
	for _, d := range []string{"a", "b", "c"} {
		e.Write("src/"+d+"/seed.php", "seed")
	}

	e.StartPoll(80)
	e.WaitForContent("dest/a/seed.php", "seed", 5*time.Second)

	var wg sync.WaitGroup
	for _, d := range []string{"a", "b", "c"} {
		d := d
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 40; i++ {
				e.Write("src/"+d+"/seed.php", d+"-final")
				time.Sleep(2 * time.Millisecond)
			}
		}()
	}
	wg.Wait()
	for _, d := range []string{"a", "b", "c"} {
		e.WaitForContent("dest/"+d+"/seed.php", d+"-final", 10*time.Second)
	}
}

// A deeply nested tree must sync completely without recursion problems.
func TestChaosDeepNesting(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Deep", "src/deep", "dest/deep"}})
	deep := "src/deep"
	for i := 0; i < 30; i++ {
		deep += "/d" + strconv.Itoa(i)
	}
	e.Write(deep+"/leaf.php", "bottom")

	e.Run("sync")

	dest := "dest/deep"
	for i := 0; i < 30; i++ {
		dest += "/d" + strconv.Itoa(i)
	}
	e.AssertContent(dest+"/leaf.php", "bottom")
}

// File names with spaces and unicode must sync correctly.
func TestChaosSpecialCharacterNames(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Weird", "src/w", "dest/w"}})
	e.Write("src/w/a file with spaces.php", "spaces")
	e.Write("src/w/ünïcödé-名前.php", "unicode")
	e.Write("src/w/with'quote.php", "quote")

	e.Run("sync")

	e.AssertContent("dest/w/a file with spaces.php", "spaces")
	e.AssertContent("dest/w/ünïcödé-名前.php", "unicode")
	e.AssertContent("dest/w/with'quote.php", "quote")
}

// A self-referential symlink must not hang or crash the tool.
func TestChaosSymlinkCycleDoesNotHang(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Cyc", "src/cyc", "dest/cyc"}})
	e.Write("src/cyc/real.php", "real")
	e.Symlink(e.Path("src/cyc/loop"), e.Path("src/cyc/loop")) // points to itself

	// Must return promptly (the harness would time out the test on a hang).
	done := make(chan struct{})
	go func() {
		_, _ = e.RunExpectFail("sync")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatalf("sync hung on a symlink cycle")
	}
}

// Rapid create/delete/recreate churn ending on a recreate must land the final
// content.
func TestChaosRapidCreateDeleteCycles(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfigOpts(utils.Options{DebounceMs: 100},
		[][3]string{{"Cyc", "src/local", "dest/local"}})
	e.Write("src/local/keep.php", "keep")

	e.StartPoll(80)
	e.WaitForContent("dest/local/keep.php", "keep", 5*time.Second)

	for i := 0; i < 15; i++ {
		e.Write("src/local/churn.php", "v"+strconv.Itoa(i))
		time.Sleep(8 * time.Millisecond)
		e.Rm("src/local/churn.php")
		time.Sleep(8 * time.Millisecond)
	}
	e.Write("src/local/churn.php", "FINAL")
	e.WaitForContent("dest/local/churn.php", "FINAL", 10*time.Second)
}

// Killing the watcher and restarting it must reconcile a change made while it
// was down (initial sync on restart).
func TestChaosWatcherRestartRecovers(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(100, [][3]string{{"Plugins", "src/local", "dest/local"}})
	e.Write("src/local/v.php", "v1")

	w1 := e.StartPoll(80)
	e.WaitForContent("dest/local/v.php", "v1", 5*time.Second)
	w1.Stop()

	e.Write("src/local/v.php", "v2-offline")

	e.StartPoll(80)
	e.WaitForContent("dest/local/v.php", "v2-offline", 5*time.Second)
}

// Watching a source that doesn't exist yet must not crash; once it appears, its
// files sync.
func TestChaosWatchMissingSourceThenCreate(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(100, [][3]string{{"Later", "src/local", "dest/local"}})
	// Note: src/local does not exist yet.

	e.StartPoll(80)

	// Create the source after the watcher is already running.
	e.Write("src/local/version.php", "v1")
	e.WaitForContent("dest/local/version.php", "v1", 8*time.Second)
}

// A large number of ignore patterns must not crash or break matching.
func TestChaosManyIgnorePatterns(t *testing.T) {
	e := utils.NewEnv(t)
	patterns := make([]string, 0, 200)
	for i := 0; i < 200; i++ {
		patterns = append(patterns, "junk"+strconv.Itoa(i)+"/**")
	}
	patterns = append(patterns, "*.log")
	e.WriteConfigOpts(utils.Options{Ignore: patterns},
		[][3]string{{"App", "src/app", "dest/app"}})
	e.Write("src/app/keep.php", "keep")
	e.Write("src/app/drop.log", "drop")
	e.Write("src/app/junk5/x.php", "drop")

	e.Run("sync")

	e.AssertContent("dest/app/keep.php", "keep")
	if e.Exists("dest/app/drop.log") || e.Exists("dest/app/junk5") {
		t.Fatalf("ignored paths should not be synced")
	}
}

// A large one-shot tree must sync completely.
func TestChaosLargeTreeOneShot(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Big", "src/big", "dest/big"}})
	const dirs, perDir = 20, 25
	for d := 0; d < dirs; d++ {
		for f := 0; f < perDir; f++ {
			e.Write("src/big/d"+strconv.Itoa(d)+"/f"+strconv.Itoa(f)+".php", strconv.Itoa(d)+"-"+strconv.Itoa(f))
		}
	}

	e.Run("sync")

	for d := 0; d < dirs; d++ {
		for f := 0; f < perDir; f++ {
			e.AssertContent("dest/big/d"+strconv.Itoa(d)+"/f"+strconv.Itoa(f)+".php", strconv.Itoa(d)+"-"+strconv.Itoa(f))
		}
	}
}

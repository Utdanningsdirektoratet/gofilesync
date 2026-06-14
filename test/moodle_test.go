// Package projects holds end-to-end tests for realistic, named project setups.
// Each file models how a specific real project configures and exercises
// gofilesync, rather than testing one feature in isolation.
package projects

import (
	"fmt"
	"testing"
	"time"

	"github.com/ajxudir/gofilesync/test/utils"
)

// moodleConfig mirrors the Moodle/docker setup: a single config file plus a set
// of plugin-type directories, synced from the source checkout into the web root.
func moodleConfig(e *utils.Env) {
	e.WriteConfig(200, [][3]string{
		{"Config", "src/.docker/config.php", "html/config.php"},
		{"Local plugins", "src/local", "html/local"},
		{"Theme plugins", "src/theme", "html/theme"},
		{"Auth plugins", "src/auth", "html/auth"},
		{"Mod plugins", "src/mod", "html/mod"},
		{"API Ping", "src/api/ping.php", "html/api/ping.php"},
	})
}

// The full Moodle developer workflow over a single polling watcher: initial
// sync, live edits, adding a new plugin directory, deleting a plugin file, and
// restoring an overwritten config from backup.
func TestMoodleDevWorkflow(t *testing.T) {
	e := utils.NewEnv(t)
	moodleConfig(e)

	// A pre-existing web-root config that the source will overwrite.
	e.Write("html/config.php", "<?php // shipped default config")
	e.Write("src/.docker/config.php", "<?php // dev config")
	e.Write("src/local/greetings/version.php", "v1")
	e.Write("src/theme/boost/config.php", "theme")
	e.Write("src/auth/oauth/auth.php", "auth")
	e.Write("src/mod/quiz/lib.php", "quiz")
	e.Write("src/api/ping.php", "pong")

	e.StartPoll(100)

	// 1. Initial sync lands every entry.
	e.WaitForContent("html/config.php", "<?php // dev config", 5*time.Second)
	e.WaitForContent("html/local/greetings/version.php", "v1", 5*time.Second)
	e.WaitForContent("html/theme/boost/config.php", "theme", 5*time.Second)
	e.WaitForContent("html/auth/oauth/auth.php", "auth", 5*time.Second)
	e.WaitForContent("html/mod/quiz/lib.php", "quiz", 5*time.Second)
	e.WaitForContent("html/api/ping.php", "pong", 5*time.Second)

	// 2. Editing a plugin file propagates.
	e.Write("src/local/greetings/version.php", "v2")
	e.WaitForContent("html/local/greetings/version.php", "v2", 5*time.Second)

	// 3. Adding a whole new plugin directory propagates (incl. nested files).
	e.Write("src/mod/assign/lib.php", "assign")
	e.Write("src/mod/assign/db/install.xml", "<xml/>")
	e.WaitForContent("html/mod/assign/lib.php", "assign", 5*time.Second)
	e.WaitForContent("html/mod/assign/db/install.xml", "<xml/>", 5*time.Second)

	// 4. Deleting a plugin file removes it at the destination.
	e.Rm("src/mod/quiz/lib.php")
	e.WaitForGone("html/mod/quiz/lib.php", 5*time.Second)

	// 5. Deleting the dev config source restores the shipped default from backup.
	e.Rm("src/.docker/config.php")
	e.WaitForContent("html/config.php", "<?php // shipped default config", 5*time.Second)
}

// A realistic editing load: many plugin files changed in quick succession must
// all converge to their final content (the kind of churn an IDE save-all or a
// git checkout produces).
func TestMoodleBulkEditConverges(t *testing.T) {
	e := utils.NewEnv(t)
	moodleConfig(e)
	const files = 30
	for i := 0; i < files; i++ {
		e.Write(fmt.Sprintf("src/local/plugin%02d/version.php", i), "init")
	}
	e.Write("src/.docker/config.php", "cfg")

	e.StartPoll(100)
	e.WaitForContent("html/local/plugin00/version.php", "init", 5*time.Second)

	for round := 0; round < 10; round++ {
		for i := 0; i < files; i++ {
			e.Write(fmt.Sprintf("src/local/plugin%02d/version.php", i), fmt.Sprintf("r%d", round))
		}
		time.Sleep(5 * time.Millisecond)
	}

	for i := 0; i < files; i++ {
		e.WaitForContent(fmt.Sprintf("html/local/plugin%02d/version.php", i), "r9", 10*time.Second)
	}
}

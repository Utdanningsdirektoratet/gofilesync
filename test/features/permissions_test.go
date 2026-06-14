package features

import (
	"runtime"
	"testing"

	"github.com/ajxudir/gofilesync/test/utils"
)

// With preservePermissions enabled, the source file's executable bit is kept.
func TestPreservePermissionsKeepsExecutableBit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission bits not meaningful on Windows")
	}
	e := utils.NewEnv(t)
	e.WriteConfigOpts(utils.Options{PreservePermissions: true},
		[][3]string{{"Script", "src/run.sh", "dest/run.sh"}})
	e.Write("src/run.sh", "#!/bin/sh\necho hi\n")
	e.Chmod("src/run.sh", 0755)

	e.Run("sync")

	if mode := e.Mode("dest/run.sh"); mode&0111 == 0 {
		t.Fatalf("expected executable bit preserved, got mode %o", mode)
	}
}

// By default (opt-in off) the executable bit is not propagated.
func TestPermissionsNotPreservedByDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission bits not meaningful on Windows")
	}
	e := utils.NewEnv(t)
	e.WriteConfig(300, [][3]string{{"Script", "src/run.sh", "dest/run.sh"}})
	e.Write("src/run.sh", "#!/bin/sh\necho hi\n")
	e.Chmod("src/run.sh", 0755)

	e.Run("sync")

	if mode := e.Mode("dest/run.sh"); mode&0111 != 0 {
		t.Fatalf("did not expect executable bit by default, got mode %o", mode)
	}
}

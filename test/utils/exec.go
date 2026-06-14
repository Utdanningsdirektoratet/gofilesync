package utils

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Run executes the binary synchronously and fails the test on non-zero exit.
func (e *Env) Run(args ...string) string {
	e.t.Helper()
	out, err := e.RunExpectFail(args...)
	if err != nil {
		e.t.Fatalf("run %v failed: %v\noutput:\n%s", args, err, out)
	}
	return out
}

// RunExpectFail executes the binary and returns output+error without failing.
func (e *Env) RunExpectFail(args ...string) (string, error) {
	e.t.Helper()
	full := append([]string{e.ConfigPath}, args...)
	out, err := exec.Command(e.bin, full...).CombinedOutput()
	return string(out), err
}

// RunRaw executes the binary with exactly the given args (no config prefix),
// returning combined output and error. Used to test top-level CLI behaviour
// like --help, --version and argument validation.
func (e *Env) RunRaw(args ...string) (string, error) {
	e.t.Helper()
	out, err := exec.Command(e.bin, args...).CombinedOutput()
	return string(out), err
}

// WatchProc is a running `watch` subprocess with captured output.
type WatchProc struct {
	cmd *exec.Cmd
	out *lockedBuffer
	env *Env
}

// StartWatch launches `watch` with optional extra args (e.g. "--poll", "200").
func (e *Env) StartWatch(extra ...string) *WatchProc {
	e.t.Helper()
	args := append([]string{e.ConfigPath, "watch"}, extra...)
	cmd := exec.Command(e.bin, args...)
	out := &lockedBuffer{}
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Start(); err != nil {
		e.t.Fatalf("start watch: %v", err)
	}
	w := &WatchProc{cmd: cmd, out: out, env: e}
	e.t.Cleanup(w.Stop)
	return w
}

// StartPoll launches `watch --poll <ms>` (the primary mode for mounted volumes
// where filesystem notifications are unreliable, e.g. Docker on macOS).
func (e *Env) StartPoll(ms int) *WatchProc {
	return e.StartWatch("--poll", fmt.Sprintf("%d", ms))
}

// Output returns everything the watch process has printed so far.
func (w *WatchProc) Output() string { return w.out.String() }

// WaitForOutput blocks until the watch output contains substr, or fails.
func (w *WatchProc) WaitForOutput(substr string, timeout time.Duration) {
	w.env.t.Helper()
	w.env.WaitFor(fmt.Sprintf("watch output contains %q", substr), timeout, func() bool {
		return strings.Contains(w.out.String(), substr)
	})
}

// Stop terminates the watch process.
func (w *WatchProc) Stop() {
	if w.cmd == nil || w.cmd.Process == nil {
		return
	}
	_ = w.cmd.Process.Kill()
	_, _ = w.cmd.Process.Wait()
	w.cmd = nil
}

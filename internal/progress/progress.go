package progress

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

// colorEnabled is false on Windows (default consoles don't reliably handle
// ANSI without virtual-terminal mode) and when not a TTY or NO_COLOR is set.
func colorEnabled(interactive bool) bool {
	return interactive && os.Getenv("NO_COLOR") == "" && runtime.GOOS != "windows"
}

func ColorEnabled(w io.Writer) bool { return colorEnabled(isInteractive(w)) }

const barWidth = 24

type Reporter struct {
	Styler
	interactive bool
	quiet       bool
	out         io.Writer
	total       int
	done        int
	current     string
	lastPct     int
	lastRender  time.Time
}

const ruleWidth = 50

func New(out io.Writer, total, entries, ignores int, quiet bool) *Reporter {
	inter := !quiet && isInteractive(out)
	r := &Reporter{
		Styler:      Styler{color: colorEnabled(inter)},
		out:         out,
		total:       total,
		lastPct:     -1,
		quiet:       quiet,
		interactive: inter,
	}
	if quiet {
		return r
	}
	sub := fmt.Sprintf("%s %s · %d %s",
		comma(total), plural(total, "file", "files"),
		entries, plural(entries, "entry", "entries"))
	if ignores > 0 {
		sub += fmt.Sprintf(" · ignoring %d %s", ignores, plural(ignores, "pattern", "patterns"))
	}
	fmt.Fprintln(out, r.rule())
	fmt.Fprintln(out, r.Bold(r.Accent("gofilesync"))+r.Dim(" · ")+r.Bold("Initial sync"))
	fmt.Fprintln(out, r.Dim(sub))
	fmt.Fprintln(out, r.rule())

	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  "+r.Dim("["+ts+"]")+"  "+r.Cyan("INFO"))
	return r
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func isInteractive(w io.Writer) bool {
	if os.Getenv("CI") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func (r *Reporter) pct() int {
	if r.total <= 0 || r.done >= r.total {
		return 100
	}
	return r.done * 100 / r.total
}

type Styler struct{ color bool }

func NewStyler(w io.Writer) Styler {
	return Styler{color: colorEnabled(isInteractive(w))}
}

func (s Styler) wrap(code, str string) string {
	if !s.color {
		return str
	}
	return "\x1b[" + code + "m" + str + "\x1b[0m"
}

func (s Styler) Bold(x string) string   { return s.wrap("1", x) }
func (s Styler) Accent(x string) string { return s.wrap("38;5;208", x) }
func (s Styler) Dim(x string) string    { return s.wrap("2", x) }
func (s Styler) Green(x string) string  { return s.wrap("32", x) }
func (s Styler) Red(x string) string    { return s.wrap("31", x) }
func (s Styler) Cyan(x string) string   { return s.wrap("36", x) }
func (s Styler) rule() string           { return s.Dim(strings.Repeat("─", ruleWidth)) }

func Banner(w io.Writer, title, subtitle string) {
	s := NewStyler(w)
	fmt.Fprintln(w, s.rule())
	fmt.Fprintln(w, s.Bold(s.Accent("gofilesync"))+s.Dim(" · ")+s.Bold(title))
	if subtitle != "" {
		fmt.Fprintln(w, s.Dim(subtitle))
	}
	fmt.Fprintln(w, s.rule())
}

func (r *Reporter) Entry(title string, isDir bool, count int) {
	r.current = title
	if r.quiet {
		return
	}
	if isDir && count == 0 {
		return
	}
	detail := "1 file"
	if isDir {
		detail = fmt.Sprintf("%s %s", comma(count), plural(count, "file", "files"))
	}
	if r.interactive {
		fmt.Fprint(r.out, "\r\x1b[K")
	}
	fmt.Fprintf(r.out, "    %s %s  %s\n", r.Accent("▸"), r.Bold(title), r.Dim("("+detail+")"))
}

func (r *Reporter) Item(name string) {
	if !r.interactive {
		return
	}
	r.current = name
	r.render(false)
}

func (r *Reporter) Add(n int) {
	r.done += n
	if r.quiet {
		return
	}
	if r.interactive {
		r.render(false)
		return
	}
	if r.total < 20 {
		return
	}
	p := r.pct()
	for _, m := range []int{25, 50, 75} {
		if r.lastPct < m && p >= m {
			fmt.Fprintln(r.out, r.Dim(fmt.Sprintf("    … %d%%", m)))
		}
	}
	if p > r.lastPct {
		r.lastPct = p
	}
}

func (r *Reporter) render(force bool) {
	p := r.pct()
	if !force && p == r.lastPct && time.Since(r.lastRender) < 80*time.Millisecond {
		return
	}
	r.lastRender = time.Now()
	r.lastPct = p
	filled := p * barWidth / 100
	bar := strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)
	line := fmt.Sprintf("[%s] %3d%%  %-20.20s %s/%s", bar, p, r.current, comma(r.done), comma(r.total))
	fmt.Fprintf(r.out, "\r%s", r.Cyan(line))
}

func (r *Reporter) Summary(synced, skipped, restored, errors int) {
	if r.interactive {
		fmt.Fprint(r.out, "\r\x1b[K")
	}
	if !r.quiet {
		fmt.Fprintln(r.out)
	}

	var parts []string
	if synced > 0 {
		parts = append(parts, fmt.Sprintf("%d synced", synced))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d unchanged", skipped))
	}
	if restored > 0 {
		parts = append(parts, fmt.Sprintf("%d restored", restored))
	}
	if errors > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", errors, plural(errors, "error", "errors")))
	}
	if len(parts) == 0 {
		parts = append(parts, "nothing to sync")
	}

	line := "[100%] Initialization complete — " + strings.Join(parts, ", ")
	if errors > 0 {
		fmt.Fprintln(r.out, "  "+r.Red(line))
	} else {
		fmt.Fprintln(r.out, "  "+r.Green(line))
	}
}

func comma(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

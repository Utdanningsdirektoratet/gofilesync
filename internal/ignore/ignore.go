package ignore

import (
	"regexp"
	"strings"
)

type pattern struct {
	re      *regexp.Regexp
	negate  bool
	dirOnly bool
}

type Matcher struct {
	patterns []pattern
}

func New(globs []string) *Matcher {
	m := &Matcher{}
	for _, g := range globs {
		g = strings.TrimSpace(g)
		if g == "" || strings.HasPrefix(g, "#") {
			continue
		}
		p := pattern{}
		if strings.HasPrefix(g, "!") {
			p.negate = true
			g = g[1:]
		}
		if strings.HasSuffix(g, "/") {
			p.dirOnly = true
			g = strings.TrimSuffix(g, "/")
		}
		g = strings.TrimSuffix(g, "/**")
		re, err := compile(g)
		if err != nil {
			continue
		}
		p.re = re
		m.patterns = append(m.patterns, p)
	}
	return m
}

func (m *Matcher) Match(relPath string, isDir bool) bool {
	if m == nil || len(m.patterns) == 0 {
		return false
	}
	relPath = strings.TrimPrefix(filepathToSlash(relPath), "./")
	ignored := false
	for _, p := range m.patterns {
		if p.dirOnly && !isDir {
			continue
		}
		if p.re.MatchString(relPath) {
			ignored = !p.negate
		}
	}
	return ignored
}

func filepathToSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

func compile(glob string) (*regexp.Regexp, error) {
	basenameOnly := !strings.Contains(glob, "/")

	var b strings.Builder
	b.WriteString("^")
	if basenameOnly {
		b.WriteString("(?:.*/)?")
	}

	runes := []rune(glob)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		switch c {
		case '*':
			if i+1 < len(runes) && runes[i+1] == '*' {
				i++
				if i+1 < len(runes) && runes[i+1] == '/' {
					i++
					b.WriteString("(?:.*/)?")
				} else {
					b.WriteString(".*")
				}
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']', '\\':
			b.WriteString("\\")
			b.WriteRune(c)
		default:
			b.WriteRune(c)
		}
	}
	b.WriteString("(?:/.*)?$")
	return regexp.Compile(b.String())
}

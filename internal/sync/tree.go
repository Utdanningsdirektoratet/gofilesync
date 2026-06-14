package sync

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajxudir/gofilesync/internal/config"
	"github.com/ajxudir/gofilesync/internal/ignore"
)

func (m *Manager) matcherFor(dir config.Directory) *ignore.Matcher {
	globs := append(append([]string{}, m.config.Ignore...), dir.Ignore...)
	return ignore.New(globs)
}

func (m *Manager) syncPath(srcPath, destPath string) error {
	srcInfo, err := os.Lstat(srcPath)
	if err != nil {
		if os.IsNotExist(err) {
			// reconcileDeletions handles a vanished source (restore/remove);
			// never touch unmanaged files here.
			m.logger.Debug(fmt.Sprintf("Source not found %s", srcPath))
			return nil
		}
		return err
	}

	if srcInfo.Mode()&os.ModeSymlink != 0 {
		resolved, err := m.fileOps.ResolveSymlink(srcPath)
		if err != nil {
			return err
		}
		srcPath = resolved
		if srcInfo, err = os.Stat(srcPath); err != nil {
			return err
		}
	}

	if srcInfo.IsDir() {
		return m.syncDirectory(srcPath, destPath)
	}

	return m.syncFile(srcPath, destPath)
}

func (m *Manager) syncDirectory(srcDir, destDir string) error {
	return walkVisible(srcDir, m.matcher,
		func(rel string) error {
			if m.dryRun {
				return nil
			}
			return os.MkdirAll(filepath.Join(destDir, rel), 0755)
		},
		func(path, rel string) error {
			if m.rep != nil {
				sub := filepath.Dir(rel)
				if sub == "." {
					sub = filepath.Base(srcDir)
				}
				m.rep.Item(sub)
			}
			return m.syncResolved(path, filepath.Join(destDir, rel))
		})
}

func (m *Manager) syncResolved(srcPath, destPath string) error {
	if info, err := os.Lstat(srcPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		resolved, rerr := m.fileOps.ResolveSymlink(srcPath)
		if rerr != nil {
			return rerr
		}
		srcPath = resolved
	}
	return m.syncFile(srcPath, destPath)
}

func (m *Manager) syncFile(srcPath, destPath string) error {
	if m.ownership != nil {
		if owner, ok := m.ownership[destPath]; ok && owner != m.curEntry {
			return nil
		}
	}

	m.managed[destPath] = true

	if destInfo, err := os.Stat(destPath); err == nil {
		if destInfo.IsDir() {
			return fmt.Errorf("destination %q is a directory but source is a file", destPath)
		}

		identical, err := m.fileOps.AreIdentical(srcPath, destPath)
		if err != nil {
			return err
		}
		if identical {
			m.result.Skipped++
			m.logger.Debug(fmt.Sprintf("Unchanged %s -> %s", srcPath, destPath))
			return nil
		}

		if err := m.backupManager.Backup(destPath); err != nil {
			return err
		}
	}

	if err := m.fileOps.Copy(srcPath, destPath); err != nil {
		return err
	}

	m.result.Synced = append(m.result.Synced, Change{Title: m.curTitle, Rel: m.relTo(destPath)})
	if m.rep != nil {
		m.rep.Add(1)
	}
	return nil
}

func (m *Manager) relTo(destPath string) string {
	if rel, ok := relUnder(m.curDestRoot, destPath); ok {
		return rel
	}
	return filepath.Base(destPath)
}

func relUnder(root, path string) (string, bool) {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return rel, true
}

func (m *Manager) keepManaged(destRoot string) {
	for p := range m.manifest.paths {
		if p == destRoot {
			m.managed[p] = true
			continue
		}
		if _, ok := relUnder(destRoot, p); ok {
			m.managed[p] = true
		}
	}
}

func (m *Manager) countPerEntry() []int {
	counts := make([]int, len(m.config.Sync))
	for i, dir := range m.config.Sync {
		counts[i] = m.countEntry(dir)
	}
	return counts
}

func (m *Manager) countEntry(dir config.Directory) int {
	info, err := os.Lstat(dir.Source)
	if err != nil {
		return 0
	}
	if !info.IsDir() {
		return 1
	}
	n := 0
	_ = walkVisible(dir.Source, m.matcherFor(dir),
		func(string) error { return nil },
		func(string, string) error { n++; return nil })
	return n
}

func walkVisible(srcDir string, matcher *ignore.Matcher, onDir func(rel string) error, onFile func(path, rel string) error) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if matcher.Match(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return onDir(rel)
		}
		return onFile(path, rel)
	})
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (m *Manager) computeOwnership() map[string]int {
	if !destsOverlap(m.config.Sync) {
		return nil
	}
	own := map[string]int{}
	for i, dir := range m.config.Sync {
		info, err := os.Lstat(dir.Source)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			own[dir.Destination] = i
			continue
		}
		_ = walkVisible(dir.Source, m.matcherFor(dir),
			func(string) error { return nil },
			func(_, rel string) error { own[filepath.Join(dir.Destination, rel)] = i; return nil })
	}
	return own
}

func destsOverlap(entries []config.Directory) bool {
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			a, b := entries[i].Destination, entries[j].Destination
			if a == b {
				return true
			}
			if _, ok := relUnder(a, b); ok {
				return true
			}
			if _, ok := relUnder(b, a); ok {
				return true
			}
		}
	}
	return false
}

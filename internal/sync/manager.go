package sync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/ajxudir/gofilesync/internal/config"
	"github.com/ajxudir/gofilesync/internal/ignore"
	"github.com/ajxudir/gofilesync/internal/logger"
	"github.com/ajxudir/gofilesync/internal/progress"
)

type Change struct {
	Title string
	Rel   string
}

type SyncResult struct {
	Synced   []Change
	Restored []Change
	Removed  []Change
	Skipped  int
	Errors   int
}

func (r *SyncResult) Changes() int {
	return len(r.Synced) + len(r.Restored) + len(r.Removed)
}

type Manager struct {
	config        *config.Config
	logger        *logger.Logger
	fileOps       *FileOperations
	backupManager *BackupManager
	dryRun        bool
	manifest      *manifest
	workers       int

	mu          sync.Mutex
	managed     map[string]bool
	matcher     *ignore.Matcher
	rep         *progress.Reporter
	result      *SyncResult
	entryCounts []int
	curTitle    string
	curDestRoot string
	curEntry    int
	ownership   map[string]int
	watchRun    bool
}

func NewManager(cfg *config.Config, log *logger.Logger, dryRun bool) (*Manager, error) {
	if !dryRun {
		if err := os.MkdirAll(cfg.GetStateDir(), 0755); err != nil {
			return nil, fmt.Errorf("cannot create state dir %q: %w", cfg.GetStateDir(), err)
		}
		if err := os.MkdirAll(cfg.GetBackupDir(), 0755); err != nil {
			return nil, fmt.Errorf("cannot create backup dir %q: %w", cfg.GetBackupDir(), err)
		}
	}

	fileOps := NewFileOperations(dryRun, cfg.PreservePermissions, cfg.Checksum)

	return &Manager{
		config:        cfg,
		logger:        log,
		fileOps:       fileOps,
		backupManager: NewBackupManager(cfg, fileOps, log),
		dryRun:        dryRun,
		manifest:      loadManifest(cfg.GetManifestFile()),
		workers:       cfg.Workers,
	}, nil
}

func (m *Manager) Init(out io.Writer, quiet bool) error {
	m.watchRun = false
	m.entryCounts = m.countPerEntry()
	total, entries := 0, 0
	for _, c := range m.entryCounts {
		total += c
		if c > 0 {
			entries++
		}
	}
	m.rep = progress.New(out, total, entries, len(m.config.Ignore), quiet)
	res, err := m.run()
	m.rep.Summary(len(res.Synced), res.Skipped, len(res.Restored), res.Errors)
	m.rep = nil
	if err != nil {
		return err
	}
	if res.Errors > 0 {
		return fmt.Errorf("%d file(s) failed to sync (see log: %s)", res.Errors, m.config.GetLogFile())
	}
	return nil
}

func (m *Manager) Sync() (*SyncResult, error) {
	m.watchRun = true
	return m.run()
}

func (m *Manager) run() (*SyncResult, error) {
	m.managed = make(map[string]bool)
	m.result = &SyncResult{}
	m.ownership = m.computeOwnership()

	for i, dir := range m.config.Sync {
		m.matcher = m.matcherFor(dir)
		m.curTitle = dir.Title
		m.curDestRoot = dir.Destination
		m.curEntry = i

		if m.watchRun && dir.Once {
			m.keepManaged(dir.Destination)
			continue
		}

		m.logger.Debug(fmt.Sprintf("Processing %s", dir.Title))

		if m.rep != nil {
			count := 0
			if i < len(m.entryCounts) {
				count = m.entryCounts[i]
			}
			m.rep.Entry(dir.Title, isDirectory(dir.Source), count)
		}

		if err := m.syncPath(dir.Source, dir.Destination); err != nil {
			m.result.Errors++
			m.logger.Error(fmt.Sprintf("Could not sync %q: %v", dir.Title, err))
		}
	}

	m.reconcileDeletions()

	m.manifest.paths = m.managed
	if !m.dryRun {
		if err := m.manifest.save(); err != nil {
			m.logger.Error(fmt.Sprintf("Could not save sync state: %v", err))
		}
	}
	return m.result, nil
}

func (m *Manager) reconcileDeletions() {
	// On errors a directory walk may have aborted part-way, leaving managed
	// entries incomplete; skip so a transient error can't mass-delete.
	if m.result.Errors > 0 {
		m.logger.Debug("Skipping deletion reconciliation due to sync errors")
		return
	}
	for p := range m.manifest.paths {
		if m.managed[p] {
			continue
		}
		kind, err := m.backupManager.Restore(p)
		if err != nil {
			m.result.Errors++
			m.logger.Error(fmt.Sprintf("Could not restore %q: %v", p, err))
			continue
		}
		title, rel := m.locate(p)
		if kind == "removed" {
			m.result.Removed = append(m.result.Removed, Change{Title: title, Rel: rel})
		} else {
			m.result.Restored = append(m.result.Restored, Change{Title: title, Rel: rel})
		}
	}
}

func (m *Manager) locate(dest string) (title, rel string) {
	for _, d := range m.config.Sync {
		if dest == d.Destination {
			return d.Title, filepath.Base(dest)
		}
		if r, ok := relUnder(d.Destination, dest); ok {
			return d.Title, r
		}
	}
	return "", dest
}

func (m *Manager) ClearCache() error {
	return m.backupManager.Clear()
}

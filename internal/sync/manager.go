package sync

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ajxudir/gofilesync/internal/config"
	"github.com/ajxudir/gofilesync/internal/logger"
)

type Manager struct {
	config        *config.Config
	logger        *logger.Logger
	fileOps       *FileOperations
	backupManager *BackupManager
	dryRun        bool
}

func NewManager(cfg *config.Config, log *logger.Logger, dryRun bool) (*Manager, error) {
	if !dryRun {
		if err := os.MkdirAll(cfg.GetStateDir(), 0755); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(cfg.GetBackupDir(), 0755); err != nil {
			return nil, err
		}
	}

	fileOps := NewFileOperations(dryRun)
	backupManager := NewBackupManager(cfg, fileOps, log)

	return &Manager{
		config:        cfg,
		logger:        log,
		fileOps:       fileOps,
		backupManager: backupManager,
		dryRun:        dryRun,
	}, nil
}

func (m *Manager) Run() error {
	for _, dir := range m.config.Directories {
		m.logger.Debug(fmt.Sprintf("Processing %s", dir.Title))
		if err := m.syncPath(dir.Source, dir.Destination); err != nil {
			m.logger.Error(fmt.Sprintf("Failed to sync %s: %v", dir.Title, err))
		}
	}
	return nil
}

func (m *Manager) syncPath(srcPath, destPath string) error {
	srcInfo, err := os.Lstat(srcPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Source doesn't exist - check if we should restore from backup
			if _, destErr := os.Stat(destPath); destErr == nil {
				// Destination exists, restore from backup
				return m.backupManager.Restore(destPath)
			}
			m.logger.Debug(fmt.Sprintf("Source not found %s", srcPath))
			return nil
		}
		return err
	}

	if srcInfo.Mode()&os.ModeSymlink != 0 {
		resolvedPath, err := m.fileOps.ResolveSymlink(srcPath)
		if err != nil {
			return err
		}
		srcPath = resolvedPath
		srcInfo, err = os.Stat(srcPath)
		if err != nil {
			return err
		}
	}

	if srcInfo.IsDir() {
		return m.syncDirectory(srcPath, destPath)
	}

	return m.syncFile(srcPath, destPath)
}

func (m *Manager) syncFile(srcPath, destPath string) error {
	destInfo, err := os.Stat(destPath)
	if err == nil {
		if destInfo.IsDir() {
			return fmt.Errorf("destination is directory but source is file: %s", destPath)
		}

		identical, err := m.fileOps.AreIdentical(srcPath, destPath)
		if err != nil {
			return err
		}

		if identical {
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

	m.logger.Info(fmt.Sprintf("Sync %s -> %s", srcPath, destPath))
	return nil
}

func (m *Manager) syncDirectory(srcDir, destDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			if !m.dryRun {
				return os.MkdirAll(destPath, 0755)
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			resolvedPath, err := m.fileOps.ResolveSymlink(path)
			if err != nil {
				return err
			}
			return m.syncFile(resolvedPath, destPath)
		}

		return m.syncFile(path, destPath)
	})
}

func (m *Manager) ClearCache() error {
	return m.backupManager.Clear()
}

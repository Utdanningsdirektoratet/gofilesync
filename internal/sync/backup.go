package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/utdanningsdirektoratet/gofilesync/internal/config"
	"github.com/utdanningsdirektoratet/gofilesync/internal/logger"
)

type BackupManager struct {
	config    *config.Config
	fileOps   *FileOperations
	logger    *logger.Logger
	backupDir string
}

func NewBackupManager(cfg *config.Config, fileOps *FileOperations, log *logger.Logger) *BackupManager {
	return &BackupManager{
		config:    cfg,
		fileOps:   fileOps,
		logger:    log,
		backupDir: cfg.GetBackupDir(),
	}
}

func (b *BackupManager) GetBackupPath(originalPath string) string {
	hash := sha256Hash(originalPath)
	filename := filepath.Base(originalPath)
	dir := fmt.Sprintf("%s_%s", sanitizePath(filepath.Dir(originalPath)), hash[:8])
	return filepath.Join(b.backupDir, dir, filename)
}

func (b *BackupManager) Backup(filePath string) error {
	backupPath := b.GetBackupPath(filePath)

	if _, err := os.Stat(backupPath); err == nil {
		b.logger.Debug("Backup already exists " + backupPath)
		return nil
	}

	if b.fileOps.dryRun {
		return nil
	}

	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	return b.fileOps.Copy(filePath, backupPath)
}

func (b *BackupManager) Restore(filePath string) (kind string, err error) {
	backupPath := b.GetBackupPath(filePath)

	if _, statErr := os.Stat(backupPath); os.IsNotExist(statErr) {
		if err := b.fileOps.Remove(filePath); err != nil && !os.IsNotExist(err) {
			return "", err
		}
		return "removed", nil
	}

	if b.fileOps.dryRun {
		return "restored", nil
	}

	if err := b.fileOps.Copy(backupPath, filePath); err != nil {
		return "", err
	}
	return "restored", nil
}

func (b *BackupManager) Clear() error {
	if b.fileOps.dryRun {
		b.logger.Info("Clear cache and backups")
		return nil
	}

	if err := os.RemoveAll(b.backupDir); err != nil {
		return err
	}

	return os.MkdirAll(b.backupDir, 0755)
}

func sanitizePath(path string) string {
	path = strings.ReplaceAll(path, "/", "_")
	path = strings.ReplaceAll(path, "\\", "_")
	path = strings.ReplaceAll(path, ":", "_")
	return path
}

func sha256Hash(text string) string {
	h := sha256.New()
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

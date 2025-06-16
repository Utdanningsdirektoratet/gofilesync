package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajxudir/gofilesync/internal/config"
	"github.com/ajxudir/gofilesync/internal/logger"
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
		b.logger.Info("Backup original " + filePath + " -> " + backupPath)
		return nil
	}

	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	if err := b.fileOps.Copy(filePath, backupPath); err != nil {
		return err
	}

	b.logger.Info("Backup original " + filePath + " -> " + backupPath)
	return nil
}

func (b *BackupManager) Restore(filePath string) error {
	backupPath := b.GetBackupPath(filePath)

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		b.fileOps.Remove(filePath)
		b.logger.Info("Deleted " + filePath + " (no backup found)")
		return nil
	}

	if b.fileOps.dryRun {
		b.logger.Info("Restore " + backupPath + " -> " + filePath)
		return nil
	}

	if err := b.fileOps.Copy(backupPath, filePath); err != nil {
		return err
	}

	b.logger.Info("Restore " + backupPath + " -> " + filePath)
	return nil
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

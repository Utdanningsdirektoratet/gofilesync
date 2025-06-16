package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

type FileOperations struct {
	dryRun bool
}

func NewFileOperations(dryRun bool) *FileOperations {
	return &FileOperations{dryRun: dryRun}
}

func (f *FileOperations) Copy(src, dst string) error {
	if f.dryRun {
		return nil
	}

	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (f *FileOperations) Remove(path string) error {
	if f.dryRun {
		return nil
	}

	return os.Remove(path)
}

func (f *FileOperations) CalculateHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (f *FileOperations) AreIdentical(src, dst string) (bool, error) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return false, err
	}

	dstInfo, err := os.Stat(dst)
	if err != nil {
		return false, err
	}

	if srcInfo.Size() != dstInfo.Size() {
		return false, nil
	}

	srcHash, err := f.CalculateHash(src)
	if err != nil {
		return false, err
	}

	dstHash, err := f.CalculateHash(dst)
	if err != nil {
		return false, err
	}

	return srcHash == dstHash, nil
}

func (f *FileOperations) ResolveSymlink(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return path, nil
	}

	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}

	return target, nil
}

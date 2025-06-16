package sync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopy(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	content := []byte("test content")
	os.WriteFile(src, content, 0644)

	fileOps := NewFileOperations(false)
	err := fileOps.Copy(src, dst)
	if err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}

	if string(result) != string(content) {
		t.Error("Content mismatch after copy")
	}
}

func TestCopyWithDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "nested", "dir", "dest.txt")

	os.WriteFile(src, []byte("content"), 0644)

	fileOps := NewFileOperations(false)
	err := fileOps.Copy(src, dst)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(dst); os.IsNotExist(err) {
		t.Error("Destination file should exist")
	}
}

func TestCopyDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	os.WriteFile(src, []byte("content"), 0644)

	fileOps := NewFileOperations(true)
	err := fileOps.Copy(src, dst)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Error("Dry run should not create file")
	}
}

func TestRemove(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(file, []byte("content"), 0644)

	fileOps := NewFileOperations(false)
	err := fileOps.Remove(file)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("File should be removed")
	}
}

func TestRemoveDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(file, []byte("content"), 0644)

	fileOps := NewFileOperations(true)
	err := fileOps.Remove(file)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		t.Error("Dry run should not remove file")
	}
}

func TestAreIdentical(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")

	os.WriteFile(file1, []byte("same content"), 0644)
	os.WriteFile(file2, []byte("same content"), 0644)
	os.WriteFile(file3, []byte("different"), 0644)

	fileOps := NewFileOperations(false)

	identical, err := fileOps.AreIdentical(file1, file2)
	if err != nil {
		t.Fatal(err)
	}
	if !identical {
		t.Error("Files should be identical")
	}

	identical, err = fileOps.AreIdentical(file1, file3)
	if err != nil {
		t.Fatal(err)
	}
	if identical {
		t.Error("Files should not be identical")
	}
}

func TestAreIdenticalDifferentSizes(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	os.WriteFile(file1, []byte("short"), 0644)
	os.WriteFile(file2, []byte("much longer content"), 0644)

	fileOps := NewFileOperations(false)

	identical, err := fileOps.AreIdentical(file1, file2)
	if err != nil {
		t.Fatal(err)
	}
	if identical {
		t.Error("Files with different sizes should not be identical")
	}
}

func TestCalculateHash(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(file, []byte("test content"), 0644)

	fileOps := NewFileOperations(false)
	hash, err := fileOps.CalculateHash(file)
	if err != nil {
		t.Fatal(err)
	}

	// Just verify we get a valid 64-character hex hash
	if len(hash) != 64 {
		t.Errorf("Expected 64-character hash, got %d characters", len(hash))
	}
}

func TestResolveSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target.txt")
	link := filepath.Join(tmpDir, "link.txt")

	os.WriteFile(target, []byte("content"), 0644)
	os.Symlink(target, link)

	fileOps := NewFileOperations(false)

	// Test symlink resolution
	resolved, err := fileOps.ResolveSymlink(link)
	if err != nil {
		t.Fatal(err)
	}

	// Clean both paths for comparison (handles /private prefix on macOS)
	resolvedClean, _ := filepath.EvalSymlinks(resolved)
	targetClean, _ := filepath.EvalSymlinks(target)

	if resolvedClean != targetClean {
		t.Errorf("Expected %s, got %s", targetClean, resolvedClean)
	}

	// Test regular file
	resolved, err = fileOps.ResolveSymlink(target)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != target {
		t.Error("Regular file should return itself")
	}
}

func TestResolveSymlinkChain(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target.txt")
	link1 := filepath.Join(tmpDir, "link1.txt")
	link2 := filepath.Join(tmpDir, "link2.txt")

	os.WriteFile(target, []byte("content"), 0644)
	os.Symlink(target, link1)
	os.Symlink(link1, link2)

	fileOps := NewFileOperations(false)
	resolved, err := fileOps.ResolveSymlink(link2)
	if err != nil {
		t.Fatal(err)
	}

	// Clean both paths for comparison (handles /private prefix on macOS)
	resolvedClean, _ := filepath.EvalSymlinks(resolved)
	targetClean, _ := filepath.EvalSymlinks(target)

	if resolvedClean != targetClean {
		t.Errorf("Should resolve to final target %s, got %s", targetClean, resolvedClean)
	}
}

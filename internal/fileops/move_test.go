package fileops

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMove_MovesFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	content := []byte("hello move")

	if err := os.WriteFile(src, content, 0600); err != nil {
		t.Fatal(err)
	}

	if err := Move(src, dst); err != nil {
		t.Fatalf("Move: %v", err)
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("src still exists after Move")
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile dst: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("dst content = %q, want %q", got, content)
	}
}

func TestMove_DestInDifferentDir(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "a")
	dstDir := filepath.Join(root, "b")

	if err := os.MkdirAll(srcDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dstDir, 0750); err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(srcDir, "file.txt")
	dst := filepath.Join(dstDir, "file.txt")
	content := []byte("cross-dir move")

	if err := os.WriteFile(src, content, 0600); err != nil {
		t.Fatal(err)
	}

	if err := Move(src, dst); err != nil {
		t.Fatalf("Move: %v", err)
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("src still exists after Move")
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile dst: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("dst content = %q, want %q", got, content)
	}
}

func TestCopyThenRemove_Basic(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	content := []byte("copy then remove")

	if err := os.WriteFile(src, content, 0600); err != nil {
		t.Fatal(err)
	}

	if err := copyThenRemove(src, dst); err != nil {
		t.Fatalf("copyThenRemove: %v", err)
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("src still exists after copyThenRemove")
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile dst: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("dst content = %q, want %q", got, content)
	}
}

func TestCopyThenRemove_PreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	if err := os.WriteFile(src, []byte("perms"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := copyThenRemove(src, dst); err != nil {
		t.Fatalf("copyThenRemove: %v", err)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(dst)
		if err != nil {
			t.Fatalf("Stat dst: %v", err)
		}
		if info.Mode().Perm() != 0755 {
			t.Errorf("dst mode = %04o, want 0755", info.Mode().Perm())
		}
	}
}

func TestCopyThenRemove_MissingSrc(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "nonexistent.txt")
	dst := filepath.Join(dir, "dst.txt")

	if err := copyThenRemove(src, dst); err == nil {
		t.Error("expected error for missing src, got nil")
	}
}

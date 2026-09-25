package main

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
)

// A refresh that keeps the installed binary's inode leaves macOS killing every
// later exec of that path, so the refresh must rename over it, not write in
// place.
func TestRefreshInstallReplacesTheBinaryItself(t *testing.T) {
	root := t.TempDir()
	installed := filepath.Join(root, ".baton", "bin")
	if err := os.MkdirAll(installed, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(installed, binaryName())
	if err := os.WriteFile(path, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	before := inodeOf(t, path)

	binRoot := filepath.Join(root, "bootstrap", ".baton", "bin")
	source := filepath.Join(binRoot, runtime.GOOS+"-"+runtime.GOARCH)
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, binaryName()), []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	sums := filepath.Join(binRoot, "SHA256SUMS")
	if err := os.WriteFile(sums, []byte("sums"), 0o644); err != nil {
		t.Fatal(err)
	}

	refreshInstall(root, binRoot, sums)

	if after := inodeOf(t, path); after == before {
		t.Fatalf("refresh kept inode %d, so it wrote in place", after)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new" {
		t.Fatalf("unexpected content: %s", content)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("unexpected mode: %s", info.Mode())
	}
	if entries, err := os.ReadDir(filepath.Dir(path)); err != nil {
		t.Fatal(err)
	} else if len(entries) != 2 {
		t.Fatalf("refresh left %d unexpected files behind", len(entries)-2)
	}
}

func inodeOf(t *testing.T, path string) uint64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Skip("inode is not available on this platform")
	}
	return uint64(stat.Ino)
}

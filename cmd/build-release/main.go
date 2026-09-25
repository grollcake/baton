package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

var targets = []struct {
	os, arch string
}{
	{"darwin", "amd64"},
	{"darwin", "arm64"},
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"windows", "amd64"},
	{"windows", "arm64"},
}

func main() {
	rootFlag := flag.String("root", ".", "repository root")
	flag.Parse()
	root, err := filepath.Abs(*rootFlag)
	if err != nil {
		fatal(err)
	}
	binRoot := filepath.Join(root, "bootstrap", ".baton", "bin")
	checksums := make([]string, 0, len(targets))
	for _, target := range targets {
		name := "baton"
		if target.os == "windows" {
			name += ".exe"
		}
		relative := filepath.Join(target.os+"-"+target.arch, name)
		output := filepath.Join(binRoot, relative)
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			fatal(err)
		}
		command := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-ldflags=-s -w", "-o", output, "./cmd/baton")
		command.Dir = root
		command.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+target.os, "GOARCH="+target.arch)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			fatal(fmt.Errorf("build %s-%s: %w", target.os, target.arch, err))
		}
		file, err := os.Open(output)
		if err != nil {
			fatal(err)
		}
		digest := sha256.New()
		if _, err := io.Copy(digest, file); err != nil {
			file.Close()
			fatal(err)
		}
		if err := file.Close(); err != nil {
			fatal(err)
		}
		checksums = append(checksums, fmt.Sprintf("%x  %s", digest.Sum(nil), filepath.ToSlash(relative)))
		fmt.Printf("built %s\n", filepath.ToSlash(relative))
	}
	sort.Strings(checksums)
	sums := filepath.Join(binRoot, "SHA256SUMS")
	if err := os.WriteFile(sums, []byte(strings.Join(checksums, "\n")+"\n"), 0o644); err != nil {
		fatal(err)
	}
	refreshInstall(root, binRoot, sums)
}

// refreshInstall updates this repository's own .baton/bin when one is present.
// Baton is installed here, and .baton/bin is not committed, so a build that
// only wrote bootstrap/.baton/bin would leave the installed binary behind its
// checksums and `baton lint` would fail until someone copied it by hand.
func refreshInstall(root, binRoot, sums string) {
	installed := filepath.Join(root, ".baton", "bin")
	if info, err := os.Stat(installed); err != nil || !info.IsDir() {
		return
	}
	name := binaryName()
	source := filepath.Join(binRoot, runtime.GOOS+"-"+runtime.GOARCH, name)
	for _, pair := range [][2]string{
		{source, filepath.Join(installed, name)},
		{sums, filepath.Join(installed, "SHA256SUMS")},
	} {
		content, err := os.ReadFile(pair[0])
		if err != nil {
			fatal(fmt.Errorf("refresh %s: %w", pair[1], err))
		}
		mode := os.FileMode(0o644)
		if pair[0] == source {
			mode = 0o755
		}
		if err := replaceFile(pair[1], content, mode); err != nil {
			fatal(fmt.Errorf("refresh %s: %w", pair[1], err))
		}
	}
	fmt.Printf("refreshed .baton/bin/%s\n", name)
}

// binaryName is the executable's file name on the host platform.
func binaryName() string {
	if runtime.GOOS == "windows" {
		return "baton.exe"
	}
	return "baton"
}

// replaceFile writes content to a new file and renames it over path, the way
// internal/baton's copyFile does. Writing over the running binary in place
// keeps its inode, and macOS then kills every later exec of that path because
// the ad-hoc signature it cached no longer matches the bytes.
func replaceFile(path string, content []byte, mode os.FileMode) error {
	temp := path + ".tmp"
	if err := os.WriteFile(temp, content, mode); err != nil {
		return err
	}
	if err := os.Rename(temp, path); err != nil {
		os.Remove(temp)
		return err
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

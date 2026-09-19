package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	if err := os.WriteFile(filepath.Join(binRoot, "SHA256SUMS"), []byte(strings.Join(checksums, "\n")+"\n"), 0o644); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

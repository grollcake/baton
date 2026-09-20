// Command make-fixture creates a throwaway installed Baton project so a
// developer or agent working on this repository can drive `new-round`
// through `gate`, `update`, `pause`, `resume`, and `remove` against a real
// install instead of composing one by hand. It never ships: it needs this
// module's source and bootstrap/, neither of which an installed project has.
//
// It has no destination argument. Its only output is one directory under the
// OS temporary directory (honouring TMPDIR), printed as key=value lines on
// stdout, and its cleanup= line is the only removal path: unlike
// `baton revert-check`, this command cannot remove its own directory, because
// the caller drives it after the process exits.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func main() {
	binaryFlag := flag.String("binary", "working", "which binary to install: working or released")
	noGit := flag.Bool("no-git", false, "skip creating a Git repository in the fixture")
	flag.Parse()
	if flag.NArg() != 0 {
		fatal(errors.New("make-fixture accepts flags only"))
	}
	if *binaryFlag != "working" && *binaryFlag != "released" {
		fatal(fmt.Errorf("--binary must be working or released, got %q", *binaryFlag))
	}

	root, err := findModuleRoot()
	if err != nil {
		fatal(err)
	}
	version, err := readVersion(root)
	if err != nil {
		fatal(err)
	}

	dir, err := os.MkdirTemp("", "baton-fixture-")
	if err != nil {
		fatal(err)
	}

	batonDir := filepath.Join(dir, ".baton")
	if err := copyBootstrapBaton(root, batonDir); err != nil {
		fatal(err)
	}
	if err := copyInstructionFiles(root, dir); err != nil {
		fatal(err)
	}
	if err := seedLog(batonDir); err != nil {
		fatal(err)
	}

	binaryPath := filepath.Join(batonDir, "bin", binaryName())
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		fatal(err)
	}
	binarySource, err := installBinary(root, binaryPath, *binaryFlag)
	if err != nil {
		fatal(err)
	}
	if err := writeChecksums(binaryPath); err != nil {
		fatal(err)
	}

	fmt.Printf("dir=%s\n", dir)
	fmt.Printf("baton_dir=%s\n", batonDir)
	fmt.Printf("binary=%s\n", binaryPath)
	fmt.Printf("binary_source=%s\n", binarySource)
	fmt.Printf("version=%s\n", version)
	if *noGit {
		fmt.Printf("git=none\n")
	} else {
		commit, err := initGitRepository(dir)
		if err != nil {
			fatal(err)
		}
		fmt.Printf("git=%s (commit %s)\n", dir, commit)
	}
	fmt.Printf("drive=BATON_DIR=%s %s\n", batonDir, binaryPath)
	fmt.Printf("cleanup=rm -rf %s\n", dir)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "baton.exe"
	}
	return "baton"
}

// findModuleRoot walks up from the working directory for the go.mod that
// declares this module, the same way App.Discover walks up for .baton.
func findModuleRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		modPath := filepath.Join(dir, "go.mod")
		if data, readErr := os.ReadFile(modPath); readErr == nil {
			if strings.Contains(string(data), "module github.com/grollcake/baton") {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("cannot locate the baton module root; run from within the repository")
		}
	}
}

func readVersion(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// copyBootstrapBaton copies bootstrap/.baton into target, excluding bin/:
// the fixture's binary is installed separately by installBinary.
func copyBootstrapBaton(root, target string) error {
	source := filepath.Join(root, "bootstrap", ".baton")
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return os.MkdirAll(target, 0o755)
		}
		if relative == "bin" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, info.Mode().Perm())
	})
}

func copyInstructionFiles(root, target string) error {
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		data, err := os.ReadFile(filepath.Join(root, "bootstrap", name))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(target, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// seedLog seeds BATON-LOG.txt with the bootstrap REQUEST/RUN_DONE pair, with
// a real timestamp in place of bootstrap's placeholder tokens.
func seedLog(batonDir string) error {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05")
	content := fmt.Sprintf(
		"%s | boot | REQUEST  | Director | Bootstrap Baton\n%s | boot | RUN_DONE | Director | Baton initialized\n",
		timestamp, timestamp,
	)
	return os.WriteFile(filepath.Join(batonDir, "BATON-LOG.txt"), []byte(content), 0o644)
}

// installBinary places the selected binary at binaryPath and reports which
// source it came from.
func installBinary(root, binaryPath, which string) (string, error) {
	if which == "released" {
		source := filepath.Join(root, "bootstrap", ".baton", "bin", runtime.GOOS+"-"+runtime.GOARCH, binaryName())
		data, err := os.ReadFile(source)
		if err != nil {
			return "", fmt.Errorf("read released binary: %w", err)
		}
		if err := os.WriteFile(binaryPath, data, 0o755); err != nil {
			return "", err
		}
		return "release-" + runtime.GOOS + "-" + runtime.GOARCH, nil
	}

	command := exec.Command("go", "build", "-o", binaryPath, "./cmd/baton")
	command.Dir = root
	command.Stdout = os.Stderr
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("build working tree binary: %w", err)
	}
	return "working-tree", nil
}

func writeChecksums(binaryPath string) error {
	file, err := os.Open(binaryPath)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	relative := filepath.ToSlash(filepath.Join(runtime.GOOS+"-"+runtime.GOARCH, binaryName()))
	line := fmt.Sprintf("%s  %s\n", digest, relative)
	return os.WriteFile(filepath.Join(filepath.Dir(binaryPath), "SHA256SUMS"), []byte(line), 0o644)
}

// initGitRepository commits the whole fixture, because a real Baton install
// commits .baton/ and remove --purge's precondition is about committed-ness.
func initGitRepository(dir string) (string, error) {
	run := func(args ...string) error {
		command := exec.Command("git", args...)
		command.Dir = dir
		command.Stdout = io.Discard
		command.Stderr = os.Stderr
		return command.Run()
	}
	if err := run("init", "--quiet"); err != nil {
		return "", fmt.Errorf("git init: %w", err)
	}
	if err := run("config", "user.email", "fixture@example.com"); err != nil {
		return "", err
	}
	if err := run("config", "user.name", "Baton Fixture"); err != nil {
		return "", err
	}
	if err := run("add", "-A"); err != nil {
		return "", fmt.Errorf("git add: %w", err)
	}
	if err := run("commit", "--quiet", "-m", "Bootstrap Baton fixture"); err != nil {
		return "", fmt.Errorf("git commit: %w", err)
	}
	output := exec.Command("git", "rev-parse", "--short", "HEAD")
	output.Dir = dir
	commit, err := output.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(commit)), nil
}

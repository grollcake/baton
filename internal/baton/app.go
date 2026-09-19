package baton

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type App struct {
	ProjectDir string
	BatonDir   string
	ConfigDir  string
	Stdout     io.Writer
	Stderr     io.Writer
	Now        func() time.Time
	Sleep      func(time.Duration)
	Random     io.Reader
	Command    func(string, ...string) ([]byte, error)
	GOOS       string
	GOARCH     string
}

func New(projectDir, batonDir string) *App {
	configDir, _ := os.UserConfigDir()
	if configDir == "" {
		configDir = os.TempDir()
	}
	return &App{
		ProjectDir: projectDir,
		BatonDir:   batonDir,
		ConfigDir:  filepath.Join(configDir, "baton"),
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Now:        time.Now,
		Sleep:      time.Sleep,
		Random:     rand.Reader,
		Command: func(name string, args ...string) ([]byte, error) {
			return exec.Command(name, args...).Output()
		},
		GOOS:   runtime.GOOS,
		GOARCH: runtime.GOARCH,
	}
}

func Discover() (*App, error) {
	if configured := os.Getenv("BATON_DIR"); configured != "" {
		batonDir, err := filepath.Abs(configured)
		if err != nil {
			return nil, err
		}
		return New(filepath.Dir(batonDir), batonDir), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		batonDir := filepath.Join(dir, ".baton")
		if info, statErr := os.Stat(batonDir); statErr == nil && info.IsDir() {
			return New(dir, batonDir), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}

	executable, err := os.Executable()
	if err == nil {
		binDir := filepath.Dir(executable)
		if filepath.Base(binDir) == "bin" {
			batonDir := filepath.Dir(binDir)
			return New(filepath.Dir(batonDir), batonDir), nil
		}
	}
	return nil, errors.New("cannot locate .baton; run from a project or set BATON_DIR")
}

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		a.usage()
		return errors.New("missing command")
	}

	command := args[0]
	args = args[1:]
	switch command {
	case "append":
		return a.runAppend(args)
	case "gate":
		return a.runGate(args)
	case "new-round":
		return a.runNewRound(args)
	case "feedback":
		return a.runFeedback(args)
	case "status":
		return a.runStatus(args)
	case "subagent-prompt", "prompt":
		return a.runPrompt(args)
	case "models":
		return a.runModels(args)
	case "check-artifact":
		return a.runCheckArtifact(args)
	case "await":
		return a.runAwait(args)
	case "lint":
		return a.Lint()
	case "merge-agent-block":
		return a.runMergeAgentBlock(args)
	case "update":
		return a.runUpdate(args)
	case "version":
		return a.runVersion(args)
	case "help", "-h", "--help":
		a.usage()
		return nil
	default:
		a.usage()
		return fmt.Errorf("unknown command: %s", command)
	}
}

func (a *App) usage() {
	fmt.Fprint(a.Stdout, `Usage:
  baton append <EVENT> --task-id <id> --role <role> --summary <text> [--path <path>]
  baton gate <before-execute|before-review|before-approval> --task-id <id>
  baton new-round <slug> --summary <text> [--branch <name>]
  baton feedback --task-id <id> --summary <text>
  baton status [--task-id <id>]
  baton prompt <plan|review|exec> --task-id <id> --key <key> [--run-number <NN>]
  baton models <list|get|set> <codex|claude-code> [model options]
  baton check-artifact <PLANNED|EXECUTED|REVIEW|CLOSE> <path> [task-id]
  baton await <PLANNED|EXECUTED|REVIEW> <path> [--task-id <id>] [--timeout <duration>]
  baton lint
  baton merge-agent-block <target-file> <source-file>
  baton update --upstream <baton-repo> [--apply]
  baton version
`)
}

func (a *App) batonPath(parts ...string) string {
	return filepath.Join(append([]string{a.BatonDir}, parts...)...)
}

func (a *App) projectPath(path string) string {
	return filepath.Join(a.ProjectDir, filepath.FromSlash(path))
}

func (a *App) runGit(args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = a.ProjectDir
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", errors.New(message)
	}
	return strings.TrimSpace(string(output)), nil
}

func binaryName(goos string) string {
	if goos == "windows" {
		return "baton.exe"
	}
	return "baton"
}

func (a *App) installedBinaryPath() string {
	return a.batonPath("bin", binaryName(a.GOOS))
}

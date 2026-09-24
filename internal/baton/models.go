package baton

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type modelChoice struct {
	Model  string `json:"model"`
	Effort string `json:"effort,omitempty"`
}

type roleModels struct {
	Director modelChoice `json:"director"`
	Planner  modelChoice `json:"planner"`
	Executor modelChoice `json:"executor"`
}

type modelPreferences map[string]roleModels

type effortChoice struct {
	Effort      string `json:"effort"`
	Description string `json:"description,omitempty"`
}

type codexCatalog struct {
	Models []struct {
		Slug                     string         `json:"slug"`
		DisplayName              string         `json:"display_name"`
		Description              string         `json:"description"`
		Visibility               string         `json:"visibility"`
		DefaultReasoningLevel    string         `json:"default_reasoning_level"`
		SupportedReasoningLevels []effortChoice `json:"supported_reasoning_levels"`
	} `json:"models"`
}

func defaultRoleModels(platform string) roleModels {
	if platform == "claude-code" {
		return roleModels{
			Director: modelChoice{Model: "opus"},
			Planner:  modelChoice{Model: "opus"},
			Executor: modelChoice{Model: "sonnet"},
		}
	}
	return roleModels{
		Director: modelChoice{Model: "gpt-5.6-sol", Effort: "high"},
		Planner:  modelChoice{Model: "gpt-5.6-sol", Effort: "xhigh"},
		Executor: modelChoice{Model: "gpt-5.6-terra", Effort: "high"},
	}
}

func validatePlatform(platform string) error {
	if platform != "codex" && platform != "claude-code" {
		return fmt.Errorf("unsupported platform: %s (codex|claude-code)", platform)
	}
	return nil
}

func (a *App) modelPreferencesPath() string {
	return filepath.Join(a.ConfigDir, "models.json")
}

func (a *App) readModelPreferences() (modelPreferences, error) {
	content, err := os.ReadFile(a.modelPreferencesPath())
	if errors.Is(err, os.ErrNotExist) {
		return modelPreferences{}, nil
	}
	if err != nil {
		return nil, err
	}
	preferences := modelPreferences{}
	if err := json.Unmarshal(content, &preferences); err != nil {
		return nil, fmt.Errorf("invalid model preferences: %w", err)
	}
	return preferences, nil
}

func (a *App) writeModelPreferences(preferences modelPreferences) error {
	content, err := json.MarshalIndent(preferences, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return atomicWrite(a.modelPreferencesPath(), content, 0o600)
}

func printJSON(writer interface{ Write([]byte) (int, error) }, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(writer, string(content))
	return err
}

// fallbackEfforts lists the efforts every recommended model supports, used
// when the catalog cannot be read and no descriptions are available.
func fallbackEfforts() []effortChoice {
	efforts := []string{"low", "medium", "high", "xhigh", "max"}
	choices := make([]effortChoice, 0, len(efforts))
	for _, effort := range efforts {
		choices = append(choices, effortChoice{Effort: effort})
	}
	return choices
}

func (a *App) listModels(platform string) error {
	if platform == "claude-code" {
		return printJSON(a.Stdout, []map[string]any{
			{"model": "opus", "description": "latest Opus model"},
			{"model": "sonnet", "description": "latest Sonnet model"},
			{"model": "haiku", "description": "latest Haiku model"},
			{"model": "inherit", "description": "inherit the Director model (subagents only)"},
		})
	}
	output, err := a.Command("codex", "debug", "models")
	if err != nil {
		fmt.Fprintf(a.Stderr, "warning: codex model discovery failed; showing recommended fallback models: %v\n", err)
		return printJSON(a.Stdout, []map[string]any{
			{"model": "gpt-5.6-sol", "efforts": fallbackEfforts()},
			{"model": "gpt-5.6-terra", "efforts": fallbackEfforts()},
			{"model": "gpt-5.6-luna", "efforts": fallbackEfforts()},
		})
	}
	var catalog codexCatalog
	if err := json.Unmarshal(output, &catalog); err != nil {
		return fmt.Errorf("invalid codex model catalog: %w", err)
	}
	models := make([]map[string]any, 0, len(catalog.Models))
	for _, model := range catalog.Models {
		if model.Visibility != "list" {
			continue
		}
		models = append(models, map[string]any{
			"model":          model.Slug,
			"display_name":   model.DisplayName,
			"description":    model.Description,
			"default_effort": model.DefaultReasoningLevel,
			"efforts":        model.SupportedReasoningLevels,
		})
	}
	return printJSON(a.Stdout, models)
}

func (a *App) runModels(args []string) error {
	parsed, err := parseArguments(args, map[string]bool{
		"--director": true, "--director-effort": true,
		"--planner": true, "--planner-effort": true,
		"--executor": true, "--executor-effort": true,
	}, nil)
	if err != nil {
		return err
	}
	if len(parsed.pos) != 2 {
		return errors.New("models requires an action and platform: <list|get|set> <codex|claude-code>")
	}
	action, platform := parsed.pos[0], parsed.pos[1]
	if err := validatePlatform(platform); err != nil {
		return err
	}
	switch action {
	case "list":
		return a.listModels(platform)
	case "get":
		preferences, err := a.readModelPreferences()
		if err != nil {
			return err
		}
		models, ok := preferences[platform]
		if !ok {
			models = defaultRoleModels(platform)
		}
		return printJSON(a.Stdout, models)
	case "set":
		for _, name := range []string{"--director", "--planner", "--executor"} {
			if strings.TrimSpace(parsed.values[name]) == "" {
				return fmt.Errorf("missing %s", name)
			}
		}
		preferences, err := a.readModelPreferences()
		if err != nil {
			return err
		}
		preferences[platform] = roleModels{
			Director: modelChoice{Model: parsed.values["--director"], Effort: parsed.values["--director-effort"]},
			Planner:  modelChoice{Model: parsed.values["--planner"], Effort: parsed.values["--planner-effort"]},
			Executor: modelChoice{Model: parsed.values["--executor"], Effort: parsed.values["--executor-effort"]},
		}
		if err := a.writeModelPreferences(preferences); err != nil {
			return err
		}
		return printJSON(a.Stdout, preferences[platform])
	default:
		return fmt.Errorf("unknown models action: %s (list|get|set)", action)
	}
}

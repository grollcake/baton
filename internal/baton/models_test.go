package baton

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestModelsGetDefaultsAndPersistPlatformChoices(t *testing.T) {
	harness := newHarness(t)
	defaults := harness.run(t, "models", "get", "codex")
	if !strings.Contains(defaults, `"model": "gpt-5.6-sol"`) {
		t.Fatalf("unexpected defaults: %s", defaults)
	}

	saved := harness.run(t, "models", "set", "codex",
		"--director", "gpt-5.6-terra", "--director-effort", "medium",
		"--planner", "gpt-5.6-sol", "--planner-effort", "xhigh",
		"--executor", "gpt-5.6-luna", "--executor-effort", "high")
	if !strings.Contains(saved, `"model": "gpt-5.6-terra"`) {
		t.Fatalf("unexpected saved choices: %s", saved)
	}

	content, err := os.ReadFile(harness.app.modelPreferencesPath())
	if err != nil {
		t.Fatal(err)
	}
	var preferences modelPreferences
	if err := json.Unmarshal(content, &preferences); err != nil {
		t.Fatal(err)
	}
	if preferences["codex"].Executor.Model != "gpt-5.6-luna" {
		t.Fatalf("unexpected preferences: %+v", preferences)
	}
	if _, ok := preferences["claude-code"]; ok {
		t.Fatal("setting codex unexpectedly changed claude-code")
	}
	if info, err := os.Stat(harness.app.modelPreferencesPath()); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("preferences mode: %v, %v", info, err)
	}
}

func TestModelsValidationAndClaudeCatalog(t *testing.T) {
	harness := newHarness(t)
	catalog := harness.run(t, "models", "list", "claude-code")
	for _, model := range []string{"opus", "sonnet", "haiku", "inherit"} {
		if !strings.Contains(catalog, `"model": "`+model+`"`) {
			t.Fatalf("catalog missing %s: %s", model, catalog)
		}
	}
	if err := harness.fail("models", "get", "unknown"); err == nil {
		t.Fatal("models accepted an unknown platform")
	}
	if err := harness.fail("models", "set", "codex", "--director", "gpt-5.6-sol"); err == nil {
		t.Fatal("models accepted incomplete role choices")
	}
	if err := harness.fail("models", "remove", "codex"); err == nil {
		t.Fatal("models accepted an unknown action")
	}
}

func TestModelsCodexCatalogAndFallback(t *testing.T) {
	harness := newHarness(t)
	harness.app.Command = func(name string, args ...string) ([]byte, error) {
		return []byte(`{"models":[
  {"slug":"shown","display_name":"Shown","description":"Shown model","visibility":"list","default_reasoning_level":"medium","supported_reasoning_levels":[{"effort":"low","description":"lighter reasoning"},{"effort":"medium"}]},
  {"slug":"hidden","display_name":"Hidden","visibility":"hide","default_reasoning_level":"medium","supported_reasoning_levels":[]}
]}`), nil
	}
	catalog := harness.run(t, "models", "list", "codex")
	if !strings.Contains(catalog, `"model": "shown"`) || strings.Contains(catalog, `"model": "hidden"`) {
		t.Fatalf("unexpected filtered catalog: %s", catalog)
	}
	for _, want := range []string{`"description": "Shown model"`, `"description": "lighter reasoning"`} {
		if !strings.Contains(catalog, want) {
			t.Fatalf("catalog dropped %s: %s", want, catalog)
		}
	}

	harness.app.Command = func(name string, args ...string) ([]byte, error) {
		return nil, errors.New("missing codex")
	}
	fallback := harness.run(t, "models", "list", "codex")
	if !strings.Contains(fallback, `"model": "gpt-5.6-sol"`) || !strings.Contains(harness.err.String(), "showing recommended fallback") {
		t.Fatalf("unexpected fallback: %s; stderr: %s", fallback, harness.err.String())
	}
}

func TestModelsRejectInvalidStoredPreferencesAndCatalog(t *testing.T) {
	harness := newHarness(t)
	if err := os.MkdirAll(harness.app.ConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness.app.modelPreferencesPath(), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := harness.fail("models", "get", "codex"); err == nil {
		t.Fatal("models accepted invalid stored preferences")
	}

	harness.app.Command = func(name string, args ...string) ([]byte, error) {
		return []byte("not json"), nil
	}
	if err := harness.fail("models", "list", "codex"); err == nil {
		t.Fatal("models accepted an invalid codex catalog")
	}
}

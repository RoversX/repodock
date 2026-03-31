package sources

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/roversx/repodock/internal/store"
)

func TestManagerDetectUsesConfiguredPaths(t *testing.T) {
	root := t.TempDir()

	codexHome := filepath.Join(root, "codex-home")
	if err := os.MkdirAll(codexHome, 0o755); err != nil {
		t.Fatalf("mkdir codex home: %v", err)
	}
	codexStatePath := filepath.Join(codexHome, ".codex-global-state.json")
	if err := os.WriteFile(codexStatePath, []byte(`{"electron-saved-workspace-roots":["/tmp/repodock"]}`), 0o644); err != nil {
		t.Fatalf("write codex state: %v", err)
	}

	claudeProjects := filepath.Join(root, "claude-projects")
	projectDir := filepath.Join(claudeProjects, "repodock")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir claude project dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "session.jsonl"), []byte("{\"cwd\":\"/tmp/repodock\"}\n"), 0o644); err != nil {
		t.Fatalf("write claude session: %v", err)
	}

	enabled := true
	cfgStore := store.NewProviderConfigStore(filepath.Join(root, "providers.json"))
	if err := cfgStore.Save(store.ProviderConfig{
		Providers: []store.ProviderEntry{
			{ID: "codex", Enabled: &enabled, Paths: []string{codexStatePath}},
			{ID: "claude", Enabled: &enabled, Paths: []string{claudeProjects}},
		},
	}); err != nil {
		t.Fatalf("save provider config: %v", err)
	}

	manager := NewManager(cfgStore)
	detections, err := manager.Detect()
	if err != nil {
		t.Fatalf("detect providers: %v", err)
	}

	if got := AvailableProviderNames(detections); len(got) != 2 || got[0] != "claude" || got[1] != "codex" {
		t.Fatalf("unexpected provider names: %#v", got)
	}

	projects, providerErrs := LoadAll(context.Background(), AvailableProviders(detections)...)
	for _, err := range providerErrs {
		t.Logf("provider error (best-effort): %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("expected merged project list of 1, got %d", len(projects))
	}
	if len(projects[0].Sources) != 2 {
		t.Fatalf("expected merged sources from codex and claude, got %#v", projects[0].Sources)
	}
}

func TestManagerDetectTracksDisabledMissingAndUnsupported(t *testing.T) {
	root := t.TempDir()

	disabled := false
	enabled := true
	cfgStore := store.NewProviderConfigStore(filepath.Join(root, "providers.json"))
	if err := cfgStore.Save(store.ProviderConfig{
		Providers: []store.ProviderEntry{
			{ID: "codex", Enabled: &disabled},
			{ID: "claude", Enabled: &enabled, Paths: []string{filepath.Join(root, "missing-claude")}},
			{ID: "mystery", Kind: "mystery", Enabled: &enabled, Paths: []string{filepath.Join(root, "somewhere")}},
		},
	}); err != nil {
		t.Fatalf("save provider config: %v", err)
	}

	manager := NewManager(cfgStore)
	detections, err := manager.Detect()
	if err != nil {
		t.Fatalf("detect providers: %v", err)
	}

	statusByID := make(map[string]Status, len(detections))
	for _, detection := range detections {
		statusByID[detection.ID] = detection.Status
	}

	if statusByID["codex"] != StatusDisabled {
		t.Fatalf("expected codex disabled, got %s", statusByID["codex"])
	}
	if statusByID["claude"] != StatusMissing {
		t.Fatalf("expected claude missing, got %s", statusByID["claude"])
	}
	if statusByID["mystery"] != StatusUnsupported {
		t.Fatalf("expected mystery unsupported, got %s", statusByID["mystery"])
	}
}

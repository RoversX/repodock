package codex

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestProviderLoadUsesSavedWorkspaceRootsOrder(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`{
		"electron-saved-workspace-roots": [
			"/tmp/repodock",
			"/tmp/apple_reminder_cli",
			"/tmp/repodock"
		],
		"active-workspace-roots": ["/tmp/repodock"],
		"project-order": ["/tmp/apple_reminder_cli"]
	}`)

	if err := os.WriteFile(filepath.Join(dir, globalStateFile), data, 0o644); err != nil {
		t.Fatalf("write global state: %v", err)
	}

	provider := NewProvider(dir)
	projects, err := provider.Load(context.Background())
	if err != nil {
		t.Fatalf("load projects: %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	if projects[0].Name != "repodock" || projects[0].Path != "/tmp/repodock" {
		t.Fatalf("unexpected first project: %#v", projects[0])
	}

	if projects[1].Name != "apple_reminder_cli" || projects[1].Path != "/tmp/apple_reminder_cli" {
		t.Fatalf("unexpected second project: %#v", projects[1])
	}
}

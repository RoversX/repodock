package claude

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProviderLoadUsesCWDFromProjectSessions(t *testing.T) {
	root := t.TempDir()
	projectsDir := filepath.Join(root, projectsDirName)
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		t.Fatalf("mkdir projects dir: %v", err)
	}

	writeProjectSession(t, projectsDir, "project-a", "older.jsonl", "/tmp/alpha-project", time.Unix(10, 0))
	writeProjectSession(t, projectsDir, "project-b", "newer.jsonl", "/tmp/beta-project", time.Unix(20, 0))
	writeProjectSession(t, projectsDir, "project-c", "dup.jsonl", "/tmp/alpha-project", time.Unix(30, 0))

	provider := NewProvider(root)
	projects, err := provider.Load(context.Background())
	if err != nil {
		t.Fatalf("load projects: %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	if projects[0].Path != "/tmp/beta-project" || projects[0].Name != "beta-project" {
		t.Fatalf("unexpected first project: %#v", projects[0])
	}

	if projects[1].Path != "/tmp/alpha-project" || projects[1].Name != "alpha-project" {
		t.Fatalf("unexpected second project: %#v", projects[1])
	}
}

func writeProjectSession(t *testing.T, projectsDir, dirName, fileName, cwd string, modTime time.Time) {
	t.Helper()

	projectDir := filepath.Join(projectsDir, dirName)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	path := filepath.Join(projectDir, fileName)
	data := []byte("{\"type\":\"user\"}\n" + "{\"cwd\":\"" + cwd + "\"}\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write session file: %v", err)
	}

	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("chtimes session file: %v", err)
	}
}

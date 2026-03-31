package vscode

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/roversx/repodock/internal/domain"
	"github.com/roversx/repodock/internal/sources/workspacestorage"
)

type Provider struct {
	storageDir string
}

func NewProvider(storageDir string) Provider {
	return Provider{storageDir: storageDir}
}

func (p Provider) Name() string { return "vscode" }

func (p Provider) Load(ctx context.Context) ([]domain.Project, error) {
	entries, err := workspacestorage.Scan(p.storageDir)
	if err != nil {
		return nil, fmt.Errorf("scan vscode workspace storage: %w", err)
	}

	projects := make([]domain.Project, 0, len(entries))
	for _, e := range entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		projects = append(projects, domain.Project{
			Name:    filepath.Base(e.Path),
			Path:    e.Path,
			Sources: []domain.Source{domain.SourceVSCode},
		})
	}
	return projects, nil
}

func WorkspaceStoragePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "Code", "User", "workspaceStorage")
	}
	return filepath.Join(home, ".config", "Code", "User", "workspaceStorage")
}

func IsAvailable() bool {
	return workspacestorage.IsAvailable(WorkspaceStoragePath())
}

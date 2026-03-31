package opencode

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/roversx/repodock/internal/domain"
)

type Provider struct {
	dbPath string
}

func NewProvider(dbPath string) Provider {
	return Provider{dbPath: dbPath}
}

func (p Provider) Name() string { return "opencode" }

func (p Provider) Load(ctx context.Context) ([]domain.Project, error) {
	query := `SELECT DISTINCT worktree FROM project WHERE worktree != '/' AND worktree != '' ORDER BY time_updated DESC`
	cmd := exec.CommandContext(ctx, "sqlite3", p.dbPath, query)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("query opencode db: %w", err)
	}

	seen := make(map[string]struct{})
	projects := make([]domain.Project, 0)

	for _, line := range strings.Split(string(out), "\n") {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		projects = append(projects, domain.Project{
			Name:    filepath.Base(path),
			Path:    path,
			Sources: []domain.Source{domain.SourceOpenCode},
		})
	}
	return projects, nil
}

func DBPath() string {
	xdgData := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if xdgData == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		xdgData = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(xdgData, "opencode", "opencode.db")
}

func IsAvailable() bool {
	path := DBPath()
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	_, err = exec.LookPath("sqlite3")
	return err == nil
}

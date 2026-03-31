package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/roversx/repodock/internal/domain"
)

const globalStateFile = ".codex-global-state.json"

type Provider struct {
	codexHome string
	statePath string
}

type globalState struct {
	SavedWorkspaceRoots  []string `json:"electron-saved-workspace-roots"`
	ActiveWorkspaceRoots []string `json:"active-workspace-roots"`
	ProjectOrder         []string `json:"project-order"`
}

func NewProvider(codexHome string) Provider {
	return Provider{codexHome: codexHome}
}

func NewProviderFromStatePath(statePath string) Provider {
	return Provider{statePath: filepath.Clean(strings.TrimSpace(statePath))}
}

func (p Provider) Name() string {
	return "codex"
}

func (p Provider) Load(ctx context.Context) ([]domain.Project, error) {
	statePath := p.globalStatePath()
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("read codex global state: %w", err)
	}

	var state globalState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode codex global state: %w", err)
	}

	projects := make([]domain.Project, 0, len(state.SavedWorkspaceRoots))
	seen := make(map[string]struct{}, len(state.SavedWorkspaceRoots))

	for _, root := range state.SavedWorkspaceRoots {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		cleaned := filepath.Clean(strings.TrimSpace(root))
		if cleaned == "" {
			continue
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}

		seen[cleaned] = struct{}{}
		projects = append(projects, domain.Project{
			Name:    filepath.Base(cleaned),
			Path:    cleaned,
			Sources: []domain.Source{domain.SourceCodex},
		})
	}

	return projects, nil
}

func (p Provider) homeDir() string {
	return HomeDir(p.codexHome)
}

func (p Provider) globalStatePath() string {
	if strings.TrimSpace(p.statePath) != "" {
		return p.statePath
	}
	return GlobalStatePath(p.codexHome)
}

func GlobalStatePath(home string) string {
	return filepath.Join(HomeDir(home), globalStateFile)
}

func IsAvailable(home string) bool {
	info, err := os.Stat(GlobalStatePath(home))
	return err == nil && !info.IsDir()
}

func HomeDir(override string) string {
	if strings.TrimSpace(override) != "" {
		return override
	}

	if value := strings.TrimSpace(os.Getenv("CODEX_HOME")); value != "" {
		return value
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ".codex"
	}

	return filepath.Join(home, ".codex")
}

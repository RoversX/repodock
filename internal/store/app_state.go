package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const stateFileName = "state.json"

type AppState struct {
	PinnedPaths    []string             `json:"pinned_paths"`
	HiddenPaths    []string             `json:"hidden_paths,omitempty"`
	ManualProjects []ManualProject      `json:"manual_projects,omitempty"`
	SortMode       string               `json:"sort_mode"`
	DisplayMode    string               `json:"display_mode"`
	LastOpened     map[string]time.Time `json:"last_opened"`
	ShowLastOpened bool                 `json:"show_last_opened"`
}

type ManualProject struct {
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}

type AppStateStore struct {
	path string
}

func NewAppStateStore(path string) AppStateStore {
	return AppStateStore{path: path}
}

func DefaultAppStateStore() AppStateStore {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return AppStateStore{path: stateFileName}
		}
		configDir = filepath.Join(home, ".repodock")
	}

	return AppStateStore{path: filepath.Join(configDir, "repodock", stateFileName)}
}

func (s AppStateStore) Load() (AppState, error) {
	if s.path == "" {
		return AppState{}, nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AppState{}, nil
		}
		return AppState{}, fmt.Errorf("read app state: %w", err)
	}

	var state AppState
	if err := json.Unmarshal(data, &state); err != nil {
		return AppState{}, fmt.Errorf("decode app state: %w", err)
	}

	slices.Sort(state.PinnedPaths)
	state.PinnedPaths = slices.Compact(state.PinnedPaths)
	slices.Sort(state.HiddenPaths)
	state.HiddenPaths = slices.Compact(state.HiddenPaths)
	state.ManualProjects = normalizeManualProjects(state.ManualProjects)
	return state, nil
}

func (s AppStateStore) Save(state AppState) error {
	if s.path == "" {
		return nil
	}

	slices.Sort(state.PinnedPaths)
	state.PinnedPaths = slices.Compact(state.PinnedPaths)
	slices.Sort(state.HiddenPaths)
	state.HiddenPaths = slices.Compact(state.HiddenPaths)
	state.ManualProjects = normalizeManualProjects(state.ManualProjects)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode app state: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create app state dir: %w", err)
	}

	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("write app state temp file: %w", err)
	}

	if err := os.Rename(tempPath, s.path); err != nil {
		return fmt.Errorf("replace app state file: %w", err)
	}

	return nil
}

func normalizeManualProjects(projects []ManualProject) []ManualProject {
	if len(projects) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(projects))
	out := make([]ManualProject, 0, len(projects))
	for _, project := range projects {
		path := filepath.Clean(strings.TrimSpace(project.Path))
		if path == "" || path == "." {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, ManualProject{
			Path: path,
			Name: strings.TrimSpace(project.Name),
		})
	}

	slices.SortFunc(out, func(a, b ManualProject) int {
		return strings.Compare(strings.ToLower(a.Path), strings.ToLower(b.Path))
	})
	return out
}

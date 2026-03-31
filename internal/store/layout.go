package store

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Pane describes one terminal pane in a layout.
type Pane struct {
	Command   string `json:"command"`    // "" = plain shell
	SplitFrom int    `json:"split_from"` // index of the pane to split from (ignored for pane 0)
	Direction string `json:"direction"`  // "right" or "down"
}

// Layout is one named layout preset for a project.
type Layout struct {
	Name  string `json:"name,omitempty"`
	Panes []Pane `json:"panes"`
}

// LayoutCollection stores every named layout for one project.
type LayoutCollection struct {
	Default string   `json:"default,omitempty"`
	Layouts []Layout `json:"layouts"`
}

// LayoutStore persists per-project layouts.
type LayoutStore struct {
	path string
}

// NewLayoutStore returns a store scoped to the given project path.
// The filename is a SHA1 hash of the canonical path to avoid collisions
// between paths like /work/foo-bar and /work/foo/bar.
func NewLayoutStore(projectPath string) LayoutStore {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return LayoutStore{path: ""}
		}
		configDir = filepath.Join(home, ".repodock")
	}
	canonical := filepath.Clean(strings.TrimSpace(projectPath))
	sum := sha1.Sum([]byte(canonical))
	filename := hex.EncodeToString(sum[:]) + ".json"
	return LayoutStore{
		path: filepath.Join(configDir, "repodock", "layouts", filename),
	}
}

func (s LayoutStore) Path() string { return s.path }

func (s LayoutStore) HasLayout() bool {
	if s.path == "" {
		return false
	}
	_, err := os.Stat(s.path)
	return err == nil
}

// Load returns the default layout for compatibility with older callers.
func (s LayoutStore) Load() (Layout, error) {
	collection, err := s.LoadCollection()
	if err != nil {
		return Layout{}, err
	}
	if len(collection.Layouts) == 0 {
		return Layout{}, nil
	}
	name := collection.Default
	if name == "" {
		name = collection.Layouts[0].Name
	}
	layout, ok := collection.Find(name)
	if ok {
		return layout, nil
	}
	return collection.Layouts[0], nil
}

// Save replaces the collection with a single default layout for compatibility.
func (s LayoutStore) Save(l Layout) error {
	name := normalizeLayoutName(l.Name)
	if name == "" {
		name = "default"
	}
	return s.SaveCollection(LayoutCollection{
		Default: name,
		Layouts: []Layout{{Name: name, Panes: normalizePanes(l.Panes)}},
	})
}

func (s LayoutStore) LoadCollection() (LayoutCollection, error) {
	if s.path == "" {
		return LayoutCollection{}, nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LayoutCollection{}, nil
		}
		return LayoutCollection{}, fmt.Errorf("read layout: %w", err)
	}

	var collection LayoutCollection
	if err := json.Unmarshal(data, &collection); err == nil && (len(collection.Layouts) > 0 || collection.Default != "") {
		return normalizeCollection(collection), nil
	}

	var legacy Layout
	if err := json.Unmarshal(data, &legacy); err == nil && len(legacy.Panes) > 0 {
		return LayoutCollection{
			Default: "default",
			Layouts: []Layout{{Name: "default", Panes: normalizePanes(legacy.Panes)}},
		}, nil
	}

	return LayoutCollection{}, nil
}

func (s LayoutStore) SaveCollection(collection LayoutCollection) error {
	if s.path == "" {
		return nil
	}
	collection = normalizeCollection(collection)
	data, err := json.MarshalIndent(collection, "", "  ")
	if err != nil {
		return fmt.Errorf("encode layout: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create layout dir: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write layout: %w", err)
	}
	return os.Rename(tmp, s.path)
}

func (s LayoutStore) Delete() error {
	if s.path == "" {
		return nil
	}
	err := os.Remove(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (c LayoutCollection) Find(name string) (Layout, bool) {
	name = normalizeLayoutName(name)
	for _, layout := range c.Layouts {
		if normalizeLayoutName(layout.Name) == name {
			return Layout{Name: normalizeLayoutName(layout.Name), Panes: normalizePanes(layout.Panes)}, true
		}
	}
	return Layout{}, false
}

func (c LayoutCollection) Names() []string {
	names := make([]string, 0, len(c.Layouts))
	for _, layout := range c.Layouts {
		names = append(names, normalizeLayoutName(layout.Name))
	}
	return names
}

func normalizeCollection(collection LayoutCollection) LayoutCollection {
	normalized := LayoutCollection{
		Default: normalizeLayoutName(collection.Default),
		Layouts: make([]Layout, 0, len(collection.Layouts)),
	}

	seen := make(map[string]struct{}, len(collection.Layouts))
	for _, layout := range collection.Layouts {
		name := normalizeLayoutName(layout.Name)
		if name == "" {
			name = "default"
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		normalized.Layouts = append(normalized.Layouts, Layout{
			Name:  name,
			Panes: normalizePanes(layout.Panes),
		})
	}

	if len(normalized.Layouts) == 0 {
		normalized.Default = ""
		return normalized
	}
	if normalized.Default == "" {
		normalized.Default = normalized.Layouts[0].Name
	}
	if _, ok := normalized.Find(normalized.Default); !ok {
		normalized.Default = normalized.Layouts[0].Name
	}

	slices.SortStableFunc(normalized.Layouts, func(a, b Layout) int {
		if a.Name == normalized.Default && b.Name != normalized.Default {
			return -1
		}
		if a.Name != normalized.Default && b.Name == normalized.Default {
			return 1
		}
		return strings.Compare(a.Name, b.Name)
	})

	return normalized
}

func normalizeLayoutName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = strings.Join(strings.Fields(name), "-")
	return strings.ToLower(name)
}

func normalizePanes(panes []Pane) []Pane {
	if len(panes) == 0 {
		return []Pane{{Command: ""}}
	}
	out := make([]Pane, len(panes))
	for i, pane := range panes {
		out[i] = pane
		if i == 0 {
			out[i].SplitFrom = 0
			out[i].Direction = ""
			continue
		}
		if out[i].Direction != "down" {
			out[i].Direction = "right"
		}
		if out[i].SplitFrom < 0 || out[i].SplitFrom >= i {
			out[i].SplitFrom = i - 1
		}
	}
	return out
}

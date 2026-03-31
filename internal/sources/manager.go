package sources

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	antigravitysource "github.com/roversx/repodock/internal/sources/antigravity"
	claudesource "github.com/roversx/repodock/internal/sources/claude"
	codexsource "github.com/roversx/repodock/internal/sources/codex"
	cursorSource "github.com/roversx/repodock/internal/sources/cursor"
	opencodesource "github.com/roversx/repodock/internal/sources/opencode"
	pisource "github.com/roversx/repodock/internal/sources/pi"
	vscodesource "github.com/roversx/repodock/internal/sources/vscode"
	"github.com/roversx/repodock/internal/store"
)

type Status string

const (
	StatusAvailable   Status = "available"
	StatusMissing     Status = "missing"
	StatusDisabled    Status = "disabled"
	StatusUnsupported Status = "unsupported"
)

type Detection struct {
	ID        string
	Kind      string
	Name      string
	Location  string
	Paths     []string
	Enabled   bool
	Available bool
	Status    Status
	Provider  Provider
}

type Manager struct {
	configStore store.ProviderConfigStore
}

type providerSpec struct {
	ID               string
	Kind             string
	Name             string
	DefaultPaths     []string
	EnabledByDefault bool
}

type providerDefinition struct {
	ID      string
	Kind    string
	Name    string
	Enabled bool
	Paths   []string
}

func NewManager(configStore store.ProviderConfigStore) Manager {
	return Manager{configStore: configStore}
}

func DefaultManager() Manager {
	return Manager{configStore: store.DefaultProviderConfigStore()}
}

func (m Manager) Detect() ([]Detection, error) {
	cfg, err := m.configStore.Load()
	if err != nil {
		return nil, err
	}

	definitions := mergeProviderDefinitions(cfg)
	detections := make([]Detection, 0, len(definitions))
	for _, definition := range definitions {
		detections = append(detections, resolveDetection(definition))
	}

	return detections, nil
}

func AvailableProviders(detections []Detection) []Provider {
	providers := make([]Provider, 0, len(detections))
	for _, detection := range detections {
		if detection.Status != StatusAvailable || detection.Provider == nil {
			continue
		}
		providers = append(providers, detection.Provider)
	}
	return providers
}

func AvailableProviderNames(detections []Detection) []string {
	names := make([]string, 0, len(detections))
	for _, detection := range detections {
		if detection.Status != StatusAvailable {
			continue
		}
		names = append(names, detection.Name)
	}

	slices.Sort(names)
	return slices.Compact(names)
}

func mergeProviderDefinitions(cfg store.ProviderConfig) []providerDefinition {
	specs := builtinProviderSpecs()
	overridesByID := make(map[string]store.ProviderEntry, len(cfg.Providers))
	extraEntries := make([]store.ProviderEntry, 0)

	for _, entry := range cfg.Providers {
		id := strings.TrimSpace(entry.ID)
		if id != "" {
			if _, ok := specs[id]; ok {
				overridesByID[id] = entry
				continue
			}
		}
		extraEntries = append(extraEntries, entry)
	}

	definitions := make([]providerDefinition, 0, len(specs)+len(extraEntries))
	for _, spec := range builtinProviderSpecList() {
		override, hasOverride := overridesByID[spec.ID]
		definitions = append(definitions, applyProviderOverride(spec, override, hasOverride))
	}

	for _, entry := range extraEntries {
		definitions = append(definitions, providerDefinitionFromEntry(entry))
	}

	return definitions
}

func builtinProviderSpecList() []providerSpec {
	return []providerSpec{
		{
			ID:               "claude",
			Kind:             "claude",
			Name:             "claude",
			DefaultPaths:     []string{claudesource.ProjectsPath("")},
			EnabledByDefault: true,
		},
		{
			ID:               "codex",
			Kind:             "codex",
			Name:             "codex",
			DefaultPaths:     []string{codexsource.GlobalStatePath("")},
			EnabledByDefault: true,
		},
		{
			ID:           "vscode",
			Kind:         "vscode",
			Name:         "vscode",
			DefaultPaths: []string{vscodesource.WorkspaceStoragePath()},
		},
		{
			ID:           "cursor",
			Kind:         "cursor",
			Name:         "cursor",
			DefaultPaths: []string{cursorSource.WorkspaceStoragePath()},
		},
		{
			ID:           "antigravity",
			Kind:         "antigravity",
			Name:         "antigravity",
			DefaultPaths: []string{antigravitysource.WorkspaceStoragePath()},
		},
		{
			ID:           "pi",
			Kind:         "pi",
			Name:         "pi",
			DefaultPaths: []string{pisource.SessionsPath()},
		},
		{
			ID:           "opencode",
			Kind:         "opencode",
			Name:         "opencode",
			DefaultPaths: []string{opencodesource.DBPath()},
		},
	}
}

func builtinProviderSpecs() map[string]providerSpec {
	specs := make(map[string]providerSpec)
	for _, spec := range builtinProviderSpecList() {
		specs[spec.ID] = spec
	}
	return specs
}

func applyProviderOverride(spec providerSpec, entry store.ProviderEntry, hasOverride bool) providerDefinition {
	definition := providerDefinition{
		ID:      spec.ID,
		Kind:    spec.Kind,
		Name:    spec.Name,
		Enabled: spec.EnabledByDefault,
		Paths:   append([]string(nil), spec.DefaultPaths...),
	}

	if !hasOverride {
		return definition
	}

	if strings.TrimSpace(entry.Kind) != "" {
		definition.Kind = strings.TrimSpace(strings.ToLower(entry.Kind))
	}
	if strings.TrimSpace(entry.Name) != "" {
		definition.Name = strings.TrimSpace(entry.Name)
	}
	if entry.Enabled != nil {
		definition.Enabled = *entry.Enabled
	}
	if len(entry.Paths) > 0 {
		definition.Paths = append([]string(nil), entry.Paths...)
	}

	return definition
}

func providerDefinitionFromEntry(entry store.ProviderEntry) providerDefinition {
	enabled := true
	if entry.Enabled != nil {
		enabled = *entry.Enabled
	}

	id := strings.TrimSpace(entry.ID)
	if id == "" {
		id = strings.TrimSpace(strings.ToLower(entry.Kind))
	}

	name := strings.TrimSpace(entry.Name)
	if name == "" {
		name = id
	}

	return providerDefinition{
		ID:      id,
		Kind:    strings.TrimSpace(strings.ToLower(entry.Kind)),
		Name:    name,
		Enabled: enabled,
		Paths:   append([]string(nil), entry.Paths...),
	}
}

func resolveDetection(def providerDefinition) Detection {
	detection := Detection{
		ID:      def.ID,
		Kind:    def.Kind,
		Name:    def.Name,
		Enabled: def.Enabled,
		Paths:   expandProviderPaths(def.Paths),
		Status:  StatusMissing,
	}

	if detection.Name == "" {
		detection.Name = detection.ID
	}

	if !def.Enabled {
		detection.Status = StatusDisabled
		return detection
	}

	if !isSupportedKind(def.Kind) {
		detection.Status = StatusUnsupported
		return detection
	}

	for _, candidate := range detection.Paths {
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}

		provider, ok := buildProvider(def.Kind, candidate, info.IsDir())
		if !ok {
			continue
		}

		detection.Location = candidate
		detection.Available = true
		detection.Status = StatusAvailable
		detection.Provider = provider
		return detection
	}

	if len(detection.Paths) > 0 {
		detection.Location = detection.Paths[0]
	}

	return detection
}

func isSupportedKind(kind string) bool {
	switch strings.TrimSpace(strings.ToLower(kind)) {
	case "claude", "codex", "vscode", "cursor", "antigravity", "pi", "opencode":
		return true
	default:
		return false
	}
}

func buildProvider(kind, location string, isDir bool) (Provider, bool) {
	switch strings.TrimSpace(strings.ToLower(kind)) {
	case "claude":
		if isDir {
			return claudesource.NewProviderFromProjectsPath(location), true
		}
		return nil, false
	case "codex":
		if isDir {
			return codexsource.NewProvider(location), true
		}
		return codexsource.NewProviderFromStatePath(location), true
	case "vscode":
		if isDir {
			return vscodesource.NewProvider(location), true
		}
		return nil, false
	case "cursor":
		if isDir {
			return cursorSource.NewProvider(location), true
		}
		return nil, false
	case "antigravity":
		if isDir {
			return antigravitysource.NewProvider(location), true
		}
		return nil, false
	case "pi":
		if isDir {
			return pisource.NewProvider(location), true
		}
		return nil, false
	case "opencode":
		if !isDir {
			return opencodesource.NewProvider(location), true
		}
		return nil, false
	default:
		return nil, false
	}
}

func expandProviderPaths(paths []string) []string {
	expanded := make([]string, 0, len(paths))
	for _, raw := range paths {
		path := strings.TrimSpace(raw)
		if path == "" {
			continue
		}

		path = os.ExpandEnv(path)
		if strings.HasPrefix(path, "~/") || path == "~" {
			home, err := os.UserHomeDir()
			if err == nil {
				if path == "~" {
					path = home
				} else {
					path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
				}
			}
		}

		expanded = append(expanded, filepath.Clean(path))
	}

	return slices.Compact(expanded)
}

func ProviderStatusSummary(detections []Detection) string {
	if len(detections) == 0 {
		return "No providers configured."
	}

	parts := make([]string, 0, len(detections))
	for _, detection := range detections {
		label := detection.Name
		if label == "" {
			label = detection.ID
		}
		parts = append(parts, fmt.Sprintf("%s:%s", label, detection.Status))
	}

	return strings.Join(parts, "  ")
}

package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const settingsFileName = "settings.json"

type Settings struct {
	Theme      ThemeSettings      `json:"theme"`
	UI         UISettings         `json:"ui"`
	Ghostty    GhosttySettings    `json:"ghostty"`
	Onboarding OnboardingSettings `json:"onboarding"`
}

type UISettings struct {
	GridWidth          string `json:"grid_width,omitempty"`
	HideProviderCounts bool   `json:"hide_provider_counts,omitempty"`
	DemoMode           bool   `json:"demo_mode,omitempty"`
}

type GhosttySettings struct {
	Open      string `json:"open,omitempty"`      // "" / "new-window"
	Layout    string `json:"layout,omitempty"`    // "" / "shell" / "dev" / "ai"
	Indicator bool   `json:"indicator,omitempty"` // show open-in-ghostty marker
}

type ThemeSettings struct {
	Family      string `json:"family,omitempty"`
	Mode        string `json:"mode,omitempty"`
	DataPalette string `json:"data_palette,omitempty"`
}

type OnboardingSettings struct {
	Seen bool `json:"seen,omitempty"`
}

type SettingsStore struct {
	path string
}

func NewSettingsStore(path string) SettingsStore {
	return SettingsStore{path: path}
}

func DefaultSettingsStore() SettingsStore {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return SettingsStore{path: settingsFileName}
		}
		configDir = filepath.Join(home, ".repodock")
	}

	return SettingsStore{path: filepath.Join(configDir, "repodock", settingsFileName)}
}

func (s SettingsStore) Path() string {
	return s.path
}

func (s SettingsStore) Load() (Settings, error) {
	if s.path == "" {
		return Settings{}, nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Settings{}, nil
		}
		return Settings{}, fmt.Errorf("read settings: %w", err)
	}

	var cfg Settings
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Settings{}, fmt.Errorf("decode settings: %w", err)
	}

	return cfg, nil
}

func (s SettingsStore) Save(cfg Settings) error {
	if s.path == "" {
		return nil
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}

	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("write settings temp file: %w", err)
	}

	if err := os.Rename(tempPath, s.path); err != nil {
		return fmt.Errorf("replace settings file: %w", err)
	}

	return nil
}

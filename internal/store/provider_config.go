package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const providerConfigFileName = "providers.json"

type ProviderConfig struct {
	Providers []ProviderEntry `json:"providers"`
}

type ProviderEntry struct {
	ID      string   `json:"id"`
	Kind    string   `json:"kind,omitempty"`
	Name    string   `json:"name,omitempty"`
	Enabled *bool    `json:"enabled,omitempty"`
	Paths   []string `json:"paths,omitempty"`
}

type ProviderConfigStore struct {
	path string
}

func NewProviderConfigStore(path string) ProviderConfigStore {
	return ProviderConfigStore{path: path}
}

func DefaultProviderConfigStore() ProviderConfigStore {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return ProviderConfigStore{path: providerConfigFileName}
		}
		configDir = filepath.Join(home, ".repodock")
	}

	return ProviderConfigStore{path: filepath.Join(configDir, "repodock", providerConfigFileName)}
}

func (s ProviderConfigStore) Path() string {
	return s.path
}

func (s ProviderConfigStore) Load() (ProviderConfig, error) {
	if s.path == "" {
		return ProviderConfig{}, nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ProviderConfig{}, nil
		}
		return ProviderConfig{}, fmt.Errorf("read provider config: %w", err)
	}

	var cfg ProviderConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ProviderConfig{}, fmt.Errorf("decode provider config: %w", err)
	}

	return cfg, nil
}

func (s ProviderConfigStore) Save(cfg ProviderConfig) error {
	if s.path == "" {
		return nil
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode provider config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create provider config dir: %w", err)
	}

	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("write provider config temp file: %w", err)
	}

	if err := os.Rename(tempPath, s.path); err != nil {
		return fmt.Errorf("replace provider config file: %w", err)
	}

	return nil
}

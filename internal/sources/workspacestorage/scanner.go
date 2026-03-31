package workspacestorage

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Entry struct {
	Path    string
	ModTime time.Time
}

type workspaceJSON struct {
	Folder    string `json:"folder"`
	Workspace string `json:"workspace"`
}

// Scan reads workspaceStorage/<hash>/workspace.json files and returns folder paths sorted by mod time.
func Scan(storageDir string) ([]Entry, error) {
	dirs, err := os.ReadDir(storageDir)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(dirs))
	seen := make(map[string]struct{})

	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}

		wsPath := filepath.Join(storageDir, d.Name(), "workspace.json")
		data, err := os.ReadFile(wsPath)
		if err != nil {
			continue
		}

		var ws workspaceJSON
		if err := json.Unmarshal(data, &ws); err != nil {
			continue
		}

		uri := strings.TrimSpace(ws.Folder)
		if uri == "" {
			uri = strings.TrimSpace(ws.Workspace)
		}
		if uri == "" {
			continue
		}

		// Only handle local file:// URIs
		if !strings.HasPrefix(uri, "file://") {
			continue
		}
		u, err := url.Parse(uri)
		if err != nil {
			continue
		}
		path := filepath.Clean(u.Path)
		if path == "" || path == "." {
			continue
		}

		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}

		modTime := time.Time{}
		if info, err := d.Info(); err == nil {
			modTime = info.ModTime()
		}

		entries = append(entries, Entry{Path: path, ModTime: modTime})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].ModTime.After(entries[j].ModTime)
	})

	return entries, nil
}

func IsAvailable(storageDir string) bool {
	info, err := os.Stat(storageDir)
	return err == nil && info.IsDir()
}

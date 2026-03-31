package activity

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LastOpened scans Claude Code and Codex session data to build a map of
// project path → most recent activity time.
func LastOpened() map[string]time.Time {
	result := make(map[string]time.Time)
	mergeClaude(result)
	mergeCodex(result)
	return result
}

func record(m map[string]time.Time, path string, t time.Time) {
	clean := filepath.Clean(path)
	if existing, ok := m[clean]; !ok || t.After(existing) {
		m[clean] = t
	}
}

// mergeClaude reads ~/.claude/projects/<encoded-path>/*.jsonl mod times.
func mergeClaude(m map[string]time.Time) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	base := filepath.Join(home, ".claude", "projects")
	entries, err := os.ReadDir(base)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Decode path: -Users-foo-bar → /Users/foo/bar
		projectPath := "/" + strings.ReplaceAll(entry.Name(), "-", "/")
		// Trim leading double slash that can appear when name starts with -
		projectPath = filepath.Clean(projectPath)

		dirPath := filepath.Join(base, entry.Name())
		files, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			info, err := f.Info()
			if err != nil {
				continue
			}
			record(m, projectPath, info.ModTime())
		}
	}
}

// mergeCodex reads ~/.codex/archived_sessions/*.jsonl, extracting cwd from
// the first line of each file.
func mergeCodex(m map[string]time.Time) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".codex", "archived_sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		modTime := info.ModTime()

		path := filepath.Join(dir, entry.Name())
		cwd := readCodexCWD(path)
		if cwd == "" {
			continue
		}
		record(m, cwd, modTime)
	}
}

// readCodexCWD reads the first line of a Codex session file and extracts cwd.
func readCodexCWD(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return ""
	}

	var meta struct {
		Payload struct {
			CWD string `json:"cwd"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(scanner.Bytes(), &meta); err != nil {
		return ""
	}
	return meta.Payload.CWD
}

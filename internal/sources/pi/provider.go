package pi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/roversx/repodock/internal/domain"
)

type Provider struct {
	sessionsDir string
}

type firstLine struct {
	CWD string `json:"cwd"`
}

func NewProvider(sessionsDir string) Provider {
	return Provider{sessionsDir: sessionsDir}
}

func (p Provider) Name() string { return "pi" }

func (p Provider) Load(ctx context.Context) ([]domain.Project, error) {
	dirs, err := os.ReadDir(p.sessionsDir)
	if err != nil {
		return nil, fmt.Errorf("read pi sessions dir: %w", err)
	}

	type candidate struct {
		cwd     string
		modTime time.Time
	}
	candidates := make([]candidate, 0, len(dirs))
	seen := make(map[string]struct{})

	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}

		cwd, modTime, err := readProjectCWD(filepath.Join(p.sessionsDir, d.Name()))
		if err != nil || cwd == "" {
			continue
		}

		if _, ok := seen[cwd]; ok {
			continue
		}
		seen[cwd] = struct{}{}
		candidates = append(candidates, candidate{cwd: cwd, modTime: modTime})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].modTime.After(candidates[j].modTime)
	})

	projects := make([]domain.Project, 0, len(candidates))
	for _, c := range candidates {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		projects = append(projects, domain.Project{
			Name:    filepath.Base(c.cwd),
			Path:    c.cwd,
			Sources: []domain.Source{domain.SourcePi},
		})
	}
	return projects, nil
}

// readProjectCWD finds the most recent .jsonl in dir and reads cwd from its first line.
func readProjectCWD(dir string) (string, time.Time, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", time.Time{}, err
	}

	type sessionFile struct {
		path    string
		modTime time.Time
	}
	var files []sessionFile
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, sessionFile{
			path:    filepath.Join(dir, e.Name()),
			modTime: info.ModTime(),
		})
	}

	sort.SliceStable(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	for _, f := range files {
		cwd, err := readFirstCWD(f.path)
		if err == nil && cwd != "" {
			return cwd, f.modTime, nil
		}
	}
	return "", time.Time{}, nil
}

func readFirstCWD(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry firstLine
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.CWD != "" {
			return filepath.Clean(entry.CWD), nil
		}
	}
	return "", scanner.Err()
}

func SessionsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pi", "agent", "sessions")
}

func IsAvailable() bool {
	info, err := os.Stat(SessionsPath())
	return err == nil && info.IsDir()
}

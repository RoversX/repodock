package claude

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

const projectsDirName = "projects"

type Provider struct {
	claudeHome   string
	projectsPath string
}

type projectCandidate struct {
	path    string
	modTime time.Time
}

type sessionEntry struct {
	CWD string `json:"cwd"`
}

func NewProvider(claudeHome string) Provider {
	return Provider{claudeHome: claudeHome}
}

func NewProviderFromProjectsPath(projectsPath string) Provider {
	return Provider{projectsPath: filepath.Clean(strings.TrimSpace(projectsPath))}
}

func (p Provider) Name() string {
	return "claude"
}

func (p Provider) Load(ctx context.Context) ([]domain.Project, error) {
	projectsDir := p.projectsDir()
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, fmt.Errorf("read claude projects: %w", err)
	}

	candidates := make([]projectCandidate, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if !entry.IsDir() {
			continue
		}

		projectPath, modTime, err := loadProjectPath(filepath.Join(projectsDir, entry.Name()))
		if err != nil {
			continue
		}
		if projectPath == "" {
			continue
		}

		projectPath = filepath.Clean(strings.TrimSpace(projectPath))
		if projectPath == "" {
			continue
		}
		if _, ok := seen[projectPath]; ok {
			continue
		}

		seen[projectPath] = struct{}{}
		candidates = append(candidates, projectCandidate{
			path:    projectPath,
			modTime: modTime,
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].modTime.After(candidates[j].modTime)
	})

	projects := make([]domain.Project, 0, len(candidates))
	for _, candidate := range candidates {
		projects = append(projects, domain.Project{
			Name:    filepath.Base(candidate.path),
			Path:    candidate.path,
			Sources: []domain.Source{domain.SourceClaude},
		})
	}

	return projects, nil
}

func ProjectsPath(home string) string {
	return filepath.Join(HomeDir(home), projectsDirName)
}

func IsAvailable(home string) bool {
	info, err := os.Stat(ProjectsPath(home))
	return err == nil && info.IsDir()
}

func (p Provider) projectsDir() string {
	if strings.TrimSpace(p.projectsPath) != "" {
		return p.projectsPath
	}
	return ProjectsPath(p.claudeHome)
}

func HomeDir(override string) string {
	if strings.TrimSpace(override) != "" {
		return override
	}

	for _, key := range []string{"CLAUDE_HOME", "CLAUDE_CONFIG_DIR"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ".claude"
	}

	return filepath.Join(home, ".claude")
}

func loadProjectPath(projectDir string) (string, time.Time, error) {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("read claude project dir: %w", err)
	}

	type sessionFile struct {
		path    string
		modTime time.Time
	}

	files := make([]sessionFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, sessionFile{
			path:    filepath.Join(projectDir, entry.Name()),
			modTime: info.ModTime(),
		})
	}

	sort.SliceStable(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	for _, file := range files {
		projectPath, err := readCWDFromJSONL(file.path)
		if err != nil {
			continue
		}
		if projectPath != "" {
			return projectPath, file.modTime, nil
		}
	}

	return "", time.Time{}, nil
}

func readCWDFromJSONL(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open claude session file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var latest string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry sessionEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if strings.TrimSpace(entry.CWD) == "" {
			continue
		}
		latest = entry.CWD
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan claude session file: %w", err)
	}

	return latest, nil
}

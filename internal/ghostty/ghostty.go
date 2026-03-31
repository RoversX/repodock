package ghostty

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/roversx/repodock/internal/store"
)

// IsRunningInGhostty reports whether the current process is inside a Ghostty terminal.
func IsRunningInGhostty() bool {
	return os.Getenv("TERM_PROGRAM") == "ghostty"
}

// OpenWindow opens a new Ghostty window at the given directory using AppleScript.
func OpenWindow(dir, title string) error {
	script := buildLayoutScript(dir, title, "shell")
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Start()
}

// OpenLayout opens a new Ghostty window with a preset split layout using AppleScript.
// layout: "shell" | "dev" (shell+claude) | "ai" (shell+claude+codex)
func OpenLayout(dir, title, layout string) error {
	script := buildLayoutScript(dir, title, layout)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Start()
}

// OpenFromLayout opens a new Ghostty window recreating a saved layout.
func OpenFromLayout(dir string, panes []store.Pane) error {
	script := buildFromLayoutScript(dir, panes)
	return exec.Command("osascript", "-e", script).Start()
}

// ReadCurrentPanes returns the process names running in each pane of the
// frontmost Ghostty window. Returns nil on any error.
func ReadCurrentPanes() []string {
	script := `
tell application "Ghostty"
	set w to window 1
	set t to selected tab of w
	set names to {}
	repeat with term in terminals of t
		set end of names to name of term
	end repeat
	return names
end tell`

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return nil
	}

	var names []string
	for _, n := range strings.Split(strings.TrimSpace(string(out)), ", ") {
		n = strings.TrimSpace(n)
		if n != "" {
			names = append(names, n)
		}
	}
	return names
}

// OpenProjects returns the set of working directories currently open in Ghostty,
// obtained via AppleScript. Returns empty map on any error or non-macOS.
func OpenProjects() map[string]struct{} {
	result := make(map[string]struct{})

	script := `
tell application "Ghostty"
	set dirs to {}
	repeat with w in windows
		repeat with t in tabs of w
			repeat with term in terminals of t
				set d to working directory of term
				if d is not missing value and d is not "" then
					set end of dirs to d
				end if
			end repeat
		end repeat
	end repeat
	return dirs
end tell`

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return result
	}

	// AppleScript returns comma-separated list
	for _, p := range strings.Split(strings.TrimSpace(string(out)), ", ") {
		p = strings.TrimSpace(p)
		if p != "" {
			result[filepath.Clean(p)] = struct{}{}
		}
	}
	return result
}

// buildFromLayoutScript generates AppleScript to recreate a saved pane layout.
func buildFromLayoutScript(dir string, panes []store.Pane) string {
	if len(panes) == 0 {
		return buildLayoutScript(dir, "", "shell")
	}

	dir = escapeAppleScript(dir)

	var sb strings.Builder
	fmt.Fprintf(&sb, `
tell application "Ghostty"
	activate
	set cfg to new surface configuration
	set initial working directory of cfg to "%s"
	set newWin to new window with configuration cfg
	delay 0.3
	tell newWin
		set t to selected tab
		set pane0 to terminal 1 of t
`, dir)

	if panes[0].Command != "" {
		fmt.Fprintf(&sb, "\t\tinput text \"%s\\n\" to pane0\n", escapeAppleScript(panes[0].Command))
		sb.WriteString("\t\tdelay 0.2\n")
	}

	for i := 1; i < len(panes); i++ {
		p := panes[i]
		direction := p.Direction
		if direction != "down" {
			direction = "right"
		}
		from := p.SplitFrom
		if from < 0 || from >= i {
			from = i - 1
		}
		fmt.Fprintf(&sb, "\t\tset pane%d to split pane%d direction %s\n", i, from, direction)
		sb.WriteString("\t\tdelay 0.2\n")
		if p.Command != "" {
			fmt.Fprintf(&sb, "\t\tinput text \"%s\\n\" to pane%d\n", escapeAppleScript(p.Command), i)
		}
	}

	sb.WriteString("\tend tell\nend tell")
	return sb.String()
}

func buildLayoutScript(dir, title, layout string) string {
	// Escape dir for AppleScript
	dir = escapeAppleScript(dir)

	switch layout {
	case "dev":
		// Left: shell, Right: claude
		return fmt.Sprintf(`
tell application "Ghostty"
	activate
	set cfg to new surface configuration
	set initial working directory of cfg to "%s"
	set newWin to new window with configuration cfg
	delay 0.3
	tell newWin
		set t to selected tab
		set rightPane to split (terminal 1 of t) direction right
		delay 0.2
		input text "claude\n" to rightPane
	end tell
end tell`, dir)

	case "ai":
		// Left: shell, Center: claude, Right: codex
		return fmt.Sprintf(`
tell application "Ghostty"
	activate
	set cfg to new surface configuration
	set initial working directory of cfg to "%s"
	set newWin to new window with configuration cfg
	delay 0.3
	tell newWin
		set t to selected tab
		set midPane to split (terminal 1 of t) direction right
		delay 0.2
		input text "claude\n" to midPane
		set rightPane to split midPane direction right
		delay 0.2
		input text "codex\n" to rightPane
	end tell
end tell`, dir)

	default:
		// shell only — just open a new window
		return fmt.Sprintf(`
tell application "Ghostty"
	activate
	set cfg to new surface configuration
	set initial working directory of cfg to "%s"
	new window with configuration cfg
end tell`, dir)
	}
}

func escapeAppleScript(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\r\n", " ",
		"\n", " ",
		"\r", " ",
		"\t", " ",
	)
	return replacer.Replace(s)
}

// PollOpenProjects polls Ghostty for open project directories every interval,
// calling onChange whenever the set changes.
func PollOpenProjects(interval time.Duration, onChange func(map[string]struct{})) chan struct{} {
	stop := make(chan struct{})
	go func() {
		prev := OpenProjects()
		onChange(prev)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				current := OpenProjects()
				if !mapsEqual(prev, current) {
					prev = current
					onChange(current)
				}
			}
		}
	}()
	return stop
}

func mapsEqual(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

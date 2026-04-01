package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/roversx/repodock/internal/activity"
	"github.com/roversx/repodock/internal/buildinfo"
	"github.com/roversx/repodock/internal/domain"
	"github.com/roversx/repodock/internal/ghostty"
	"github.com/roversx/repodock/internal/sources"
	"github.com/roversx/repodock/internal/store"
	uitheme "github.com/roversx/repodock/internal/theme"
)

const (
	cardHeight       = 5
	sectionGapHeight = 1
)

type uiStyles struct {
	page            lipgloss.Style
	header          lipgloss.Style
	headerVersion   lipgloss.Style
	subtle          lipgloss.Style
	inputShell      lipgloss.Style
	input           lipgloss.Style
	inputPrompt     lipgloss.Style
	command         lipgloss.Style
	card            lipgloss.Style
	selectedCard    lipgloss.Style
	projectName     lipgloss.Style
	path            lipgloss.Style
	source          lipgloss.Style
	status          lipgloss.Style
	viewChip        lipgloss.Style
	activeViewChip  lipgloss.Style
	empty           lipgloss.Style
	listIndicator   lipgloss.Style
	listSelected    lipgloss.Style
	listSelectedAlt lipgloss.Style
	listNormal      lipgloss.Style
	listNormalAlt   lipgloss.Style
	panel           lipgloss.Style
	panelTitle      lipgloss.Style
	panelSubtitle   lipgloss.Style
	panelSelected   lipgloss.Style
}

var (
	activeTheme   = uitheme.Default()
	currentStyles = buildUIStyles(activeTheme.Palette)
)

func buildUIStyles(p uitheme.Palette) uiStyles {
	return uiStyles{
		page: lipgloss.NewStyle().Padding(1, 3),
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(p.Accent)),
		headerVersion: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.TextSubtle)),
		subtle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.TextSubtle)),
		inputShell: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(p.Border)).
			Padding(0, 1),
		input: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.Text)),
		inputPrompt: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.Placeholder)),
		command: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.TextMuted)),
		card: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(p.Border)).
			Padding(0, 2),
		selectedCard: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(p.BorderActive)).
			Padding(0, 2),
		projectName: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(p.Text)),
		path: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.TextSubtle)),
		source: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.AccentAlt)),
		status: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.TextMuted)),
		viewChip: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.TextMuted)).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(p.Border)).
			Padding(0, 1),
		activeViewChip: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.Text)).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(p.BorderActive)).
			Padding(0, 1),
		empty: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.TextSubtle)),
		listIndicator: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.Accent)).
			Bold(true),
		listSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.Text)),
		listSelectedAlt: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.TextMuted)),
		listNormal: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.TextMuted)),
		listNormalAlt: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.TextSubtle)),
		panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(p.Border)).
			Padding(0, 1),
		panelTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(p.Accent)),
		panelSubtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.TextMuted)),
		panelSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(p.Accent)),
	}
}

func applyTheme(resolved uitheme.Resolved) {
	activeTheme = resolved
	currentStyles = buildUIStyles(resolved.Palette)
}

func applyInputStyles(input *textinput.Model) {
	input.TextStyle = currentStyles.input
	input.PromptStyle = currentStyles.inputPrompt
	input.Cursor.Style = currentStyles.input
}

type commandEntry struct {
	name string
	desc string
}

var allCommands = []commandEntry{
	{"help", "show available commands"},
	{"demo", "toggle privacy-safe demo projects"},
	{"onboard", "show the onboarding guide"},
	{"providers", "show detected project providers"},
	{"view", "switch project view: /view mixed|provider"},
	{"layout", "open layout editor for selected project"},
	{"theme", "open theme browser or set family: /theme tokyonight"},
	{"mode", "set theme mode: /mode auto|dark|light"},
	{"palette", "set data palette: /palette tableau10"},
	{"grid", "switch to grid display"},
	{"list", "switch to list display"},
	{"settings", "open settings panel"},
	{"sync", "sync projects from providers"},
	{"shell", "open shell in selected project"},
	{"reveal", "open selected project in Finder: /reveal [reveal|open]"},
	{"new", "open the manual project import panel"},
	{"rename", "rename selected project: /rename New Name"},
	{"pin", "pin selected project to top"},
	{"unpin", "remove selected project from top pins"},
	{"hide", "hide selected project from repodock"},
	{"hidden", "show hidden projects"},
	{"unhide-all", "restore all hidden projects"},
	{"copy", "copy selected project path"},
	{"codex", "launch codex in selected project"},
	{"claude", "launch claude in selected project"},
	{"code", "open selected project in VS Code"},
	{"cursor", "open selected project in Cursor"},
	{"clear", "clear status line"},
}

type actionEntry struct {
	id    string
	label string
	desc  string
}

type onboardingSource struct {
	ID        string
	Name      string
	Enabled   bool
	Available bool
	Status    sources.Status
}

type demoProjectSeed struct {
	Name    string
	RelPath string
	Sources []domain.Source
	Opened  time.Duration
}

type model struct {
	cursor                int
	cmdCursor             int
	actionCursor          int
	rowOffset             int
	viewFocus             bool
	viewFocusCursor       int
	hoverViewMode         string
	projects              []domain.Project
	projectsBase          []domain.Project
	manualProjects        map[string]store.ManualProject
	pinnedPaths           map[string]struct{}
	hiddenPaths           map[string]struct{}
	input                 textinput.Model
	status                string
	width                 int
	height                int
	actionOpen            bool
	stateStore            store.AppStateStore
	settingsStore         store.SettingsStore
	providerConfigStore   store.ProviderConfigStore
	settings              store.Settings
	providers             []sources.Detection
	viewMode              string
	displayMode           string // "grid" or "list"
	sortMode              string // "name" or "recent"
	loadingProjects       bool
	loadingProjectsFrame  int
	lastOpened            map[string]time.Time
	showLastOpened        bool
	ghosttyOpen           map[string]struct{} // paths currently open in Ghostty
	settingsOpen          bool
	settingsCursor        int
	hiddenOpen            bool
	hiddenCursor          int
	newProjectOpen        bool
	newProjectCursor      int
	newProjectPathInput   textinput.Model
	newProjectNameInput   textinput.Model
	onboardingOpen        bool
	onboardingIsProviders bool // true when opened via /providers (not first-run)
	onboardingCursor      int
	onboardingDisplay     string
	onboardingSources     []onboardingSource
	layoutOpen            bool
	layoutDetailOpen      bool
	layoutProject         domain.Project
	layouts               []store.Layout
	layoutDefault         string
	layoutSelected        int
	layoutName            string
	layoutPanes           []store.Pane
	layoutCursor          int
	layoutField           int  // 0=command 1=direction 2=splitFrom
	layoutEditing         bool // command text input active
	layoutInput           textinput.Model
	layoutNaming          bool
	layoutNameInput       textinput.Model
	layoutHasSaved        bool
	layoutDirty           bool
	layoutSource          string
	layoutDeleteConfirm   bool
	resolvedTheme         uitheme.Resolved
	themePickerOpen       bool
	themePickerCursor     int
	themePickerOriginal   store.ThemeSettings
}

type projectsLoadedMsg struct {
	projects       []domain.Project
	manualProjects map[string]store.ManualProject
	pinnedPaths    map[string]struct{}
	hiddenPaths    map[string]struct{}
	providers      []sources.Detection
	providerErrs   []error
	sortMode       string
	displayMode    string
	lastOpened     map[string]time.Time
	showLastOpened bool
}

type projectsLoadFailedMsg struct {
	err       error
	providers []sources.Detection
}

type stateSavedMsg struct {
	action string
}

type stateSaveFailedMsg struct {
	err error
}

type shellExitedMsg struct {
	project string
	err     error
}

type projectLaunchTickMsg struct{}

type ghosttyOpenChangedMsg struct {
	paths map[string]struct{}
}

type copyPathFinishedMsg struct {
	path string
	err  error
}

type settingsSavedMsg struct {
	action   string
	settings store.Settings
}

type settingsSaveFailedMsg struct {
	err error
}

type providerConfigSavedMsg struct {
	reload bool
}

type providerConfigSaveFailedMsg struct {
	err error
}

type layoutGrabbedMsg struct {
	panes []store.Pane
}

type layoutSavedMsg struct {
	project string
	name    string
	auto    bool
	err     error
}

type layoutAppliedMsg struct {
	project string
	err     error
}

type layoutDeletedMsg struct {
	project string
	name    string
	err     error
}

type layoutDefaultSetMsg struct {
	project string
	name    string
	err     error
}

type revealInFinderFinishedMsg struct {
	path string
	mode string
	err  error
}

func (m model) settingsRowCount() int {
	return 12
}

func Run() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

func initialModel() model {
	settingsStore := store.DefaultSettingsStore()
	settings, _ := settingsStore.Load()
	providerConfigStore := store.DefaultProviderConfigStore()
	resolvedTheme := uitheme.Resolve(settings.Theme)
	applyTheme(resolvedTheme)

	input := textinput.New()
	input.Placeholder = "Type command or search project"
	input.Prompt = ""
	input.CharLimit = 256
	input.Focus()
	applyInputStyles(&input)

	layoutInput := textinput.New()
	layoutInput.Placeholder = "command (empty = shell)"
	layoutInput.Prompt = ""
	layoutInput.CharLimit = 128

	layoutNameInput := textinput.New()
	layoutNameInput.Placeholder = "layout name"
	layoutNameInput.Prompt = ""
	layoutNameInput.CharLimit = 64

	newProjectPathInput := textinput.New()
	newProjectPathInput.Placeholder = "drag folder here or paste absolute path"
	newProjectPathInput.Prompt = ""
	newProjectPathInput.CharLimit = 512

	newProjectNameInput := textinput.New()
	newProjectNameInput.Placeholder = "display name (optional)"
	newProjectNameInput.Prompt = ""
	newProjectNameInput.CharLimit = 128

	return model{
		projects:            nil,
		projectsBase:        nil,
		manualProjects:      make(map[string]store.ManualProject),
		pinnedPaths:         make(map[string]struct{}),
		hiddenPaths:         make(map[string]struct{}),
		input:               input,
		status:              "Loading project providers...",
		stateStore:          store.DefaultAppStateStore(),
		settingsStore:       settingsStore,
		providerConfigStore: providerConfigStore,
		settings:            settings,
		viewMode:            "mixed",
		displayMode:         "grid",
		sortMode:            "name",
		loadingProjects:     true,
		lastOpened:          make(map[string]time.Time),
		onboardingOpen:      !settings.Onboarding.Seen,
		onboardingDisplay:   "grid",
		resolvedTheme:       resolvedTheme,
		layoutInput:         layoutInput,
		layoutNameInput:     layoutNameInput,
		newProjectPathInput: newProjectPathInput,
		newProjectNameInput: newProjectNameInput,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, loadProjectsCmd(m.stateStore), projectLaunchTickCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncInputWidth()
		m.syncViewport()
		return m, nil
	case projectsLoadedMsg:
		m.loadingProjects = false
		m.loadingProjectsFrame = 0
		projects := msg.projects
		providers := msg.providers
		lastOpened := msg.lastOpened
		if m.demoMode() {
			projects, providers, lastOpened = demoDataset()
		}
		m.projectsBase = projects
		m.manualProjects = msg.manualProjects
		if m.manualProjects == nil {
			m.manualProjects = make(map[string]store.ManualProject)
		}
		m.pinnedPaths = msg.pinnedPaths
		m.hiddenPaths = msg.hiddenPaths
		m.providers = providers
		if msg.sortMode != "" {
			m.sortMode = msg.sortMode
		}
		if msg.displayMode != "" {
			m.displayMode = msg.displayMode
		}
		if lastOpened != nil {
			m.lastOpened = lastOpened
		}
		m.showLastOpened = msg.showLastOpened
		m.ensureViewModeValid()
		m.projects = sortAndOrderProjects(projects, msg.pinnedPaths, m.sortMode, m.lastOpened)
		m.syncOnboardingState()
		m.syncCursor()
		m.actionOpen = false
		m.status = loadedProjectsStatus(len(m.projects), providers, msg.providerErrs)
		if m.demoMode() {
			m.status = fmt.Sprintf("Demo mode on. Showing %d privacy-safe projects.", len(m.projects))
		}
		if m.settings.Ghostty.Indicator && ghostty.IsRunningInGhostty() {
			return m, queryGhosttyOpenCmd()
		}
		return m, nil
	case projectsLoadFailedMsg:
		m.loadingProjects = false
		m.loadingProjectsFrame = 0
		if m.demoMode() {
			projects, providers, lastOpened := demoDataset()
			m.providers = providers
			m.projectsBase = projects
			m.lastOpened = lastOpened
			m.ensureViewModeValid()
			m.projects = sortAndOrderProjects(projects, m.pinnedPaths, m.sortMode, m.lastOpened)
			m.status = fmt.Sprintf("Demo mode on. Showing %d privacy-safe projects.", len(m.projects))
			return m, nil
		}
		m.providers = msg.providers
		m.projects = nil
		m.projectsBase = nil
		m.status = fmt.Sprintf("Provider load failed: %v", msg.err)
		return m, nil
	case stateSavedMsg:
		if msg.action != "" {
			m.status = msg.action
		}
		return m, nil
	case stateSaveFailedMsg:
		m.status = fmt.Sprintf("Failed to save state: %v", msg.err)
		return m, nil
	case settingsSavedMsg:
		m.settings = msg.settings
		m.status = msg.action
		return m, nil
	case settingsSaveFailedMsg:
		m.status = fmt.Sprintf("Failed to save settings: %v", msg.err)
		return m, nil
	case providerConfigSavedMsg:
		if msg.reload {
			m.loadingProjects = true
			m.loadingProjectsFrame = 0
			return m, tea.Batch(loadProjectsCmd(m.stateStore), projectLaunchTickCmd())
		}
		return m, nil
	case providerConfigSaveFailedMsg:
		m.status = fmt.Sprintf("Failed to save providers: %v", msg.err)
		return m, nil
	case layoutGrabbedMsg:
		m.layoutPanes = msg.panes
		m.layoutCursor = 0
		m.layoutField = 0
		m.layoutDirty = true
		m.layoutSource = "ghostty"
		m.status = fmt.Sprintf("Grabbed %d panes from Ghostty.", len(msg.panes))
		return m, m.autosaveLayoutCmd()
	case layoutSavedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Layout save failed: %v", msg.err)
		} else {
			m.reloadLayouts(msg.name)
			if !msg.auto {
				m.status = fmt.Sprintf("Saved layout %s for %s.", msg.name, msg.project)
			}
		}
		return m, nil
	case layoutAppliedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Layout apply failed: %v", msg.err)
		} else {
			m.status = fmt.Sprintf("Applied layout for %s.", msg.project)
		}
		return m, nil
	case layoutDeletedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Layout delete failed: %v", msg.err)
		} else {
			m.layoutDeleteConfirm = false
			m.reloadLayouts("")
			m.status = fmt.Sprintf("Deleted layout %s for %s.", msg.name, msg.project)
		}
		return m, nil
	case layoutDefaultSetMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Set default layout failed: %v", msg.err)
		} else {
			m.reloadLayouts(msg.name)
			m.status = fmt.Sprintf("Default layout set to %s for %s.", msg.name, msg.project)
		}
		return m, nil
	case revealInFinderFinishedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Reveal in Finder failed: %v", msg.err)
		} else {
			switch msg.mode {
			case "open":
				m.status = fmt.Sprintf("Opened %s in Finder.", filepath.Base(msg.path))
			default:
				m.status = fmt.Sprintf("Revealed %s in Finder.", filepath.Base(msg.path))
			}
		}
		return m, nil
	case shellExitedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Shell closed with error for %s: %v", msg.project, msg.err)
			return m, tea.EnableMouseCellMotion
		}
		m.status = fmt.Sprintf("Returned from shell in %s.", msg.project)
		return m, tea.EnableMouseCellMotion
	case projectLaunchTickMsg:
		if !m.loadingProjects {
			return m, nil
		}
		m.loadingProjectsFrame++
		return m, projectLaunchTickCmd()
	case ghosttyOpenChangedMsg:
		m.ghosttyOpen = msg.paths
		return m, nil
	case copyPathFinishedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Copy path failed: %v", msg.err)
			return m, nil
		}
		m.status = fmt.Sprintf("Copied path: %s", msg.path)
		return m, nil
	case tea.MouseMsg:
		if m.loadingProjects {
			return m, nil
		}
		if m.hiddenOpen {
			return m, nil
		}
		return m.handleMouse(msg)
	case tea.KeyMsg:
		// Filter leaked terminal protocol sequences that arrive as key events.
		if isLeakedTerminalSequence(msg.String()) {
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q":
			if m.onboardingOpen {
				return m, m.closeOnboarding()
			}
			if m.newProjectOpen {
				m.closeNewProjectImport()
				return m, nil
			}
			if m.themePickerOpen {
				return m, m.closeThemePicker(false)
			}
			if m.settingsOpen {
				m.settingsOpen = false
				return m, tea.Batch(
					saveStateCmd(m.stateStore, m.buildAppState(), ""),
					saveSettingsCmd(m.settingsStore, m.settings, ""),
				)
			}
			if m.hiddenOpen {
				m.hiddenOpen = false
				m.hiddenCursor = 0
				m.status = "Hidden projects closed."
				return m, nil
			}
			if m.layoutOpen {
				m.layoutOpen = false
				m.layoutDetailOpen = false
				m.layoutNaming = false
				m.layoutDeleteConfirm = false
				return m, nil
			}
			return m, tea.Quit
		case "esc":
			if m.onboardingOpen {
				return m, m.closeOnboarding()
			}
			if m.newProjectOpen {
				m.closeNewProjectImport()
				return m, nil
			}
			if m.themePickerOpen {
				return m, m.closeThemePicker(false)
			}
			if m.settingsOpen {
				m.settingsOpen = false
				m.status = "Settings closed."
				return m, tea.Batch(
					saveStateCmd(m.stateStore, m.buildAppState(), ""),
					saveSettingsCmd(m.settingsStore, m.settings, ""),
				)
			}
			if m.hiddenOpen {
				m.hiddenOpen = false
				m.hiddenCursor = 0
				m.status = "Hidden projects closed."
				return m, nil
			}
			if m.layoutOpen {
				if m.layoutNaming {
					m.layoutNaming = false
					m.layoutNameInput.Blur()
					m.layoutNameInput.SetValue("")
					m.status = fmt.Sprintf("Layout summary: %s.", m.layoutProject.Name)
					return m, nil
				}
				if m.layoutDeleteConfirm {
					m.layoutDeleteConfirm = false
					m.status = fmt.Sprintf("Layout summary: %s.", m.layoutProject.Name)
					return m, nil
				}
				if m.layoutEditing {
					m.layoutEditing = false
					m.layoutInput.Blur()
					return m, nil
				}
				if m.layoutDetailOpen {
					m.layoutDetailOpen = false
					m.status = fmt.Sprintf("Layout summary: %s.", m.layoutProject.Name)
					return m, nil
				}
				m.layoutOpen = false
				m.layoutDetailOpen = false
				return m, nil
			}
			if m.viewFocus {
				m.viewFocus = false
				return m, nil
			}
			if m.actionOpen {
				m.actionOpen = false
				m.actionCursor = 0
				m.status = "Closed actions."
				return m, nil
			}
			m.input.SetValue("")
			m.status = "Cleared command bar."
			m.syncCursor()
			return m, nil
		case "tab":
			if m.loadingProjects {
				return m, nil
			}
			if m.newProjectOpen {
				m.newProjectCursor = (m.newProjectCursor + 1) % 2
				m.syncNewProjectFocus()
				return m, nil
			}
			if !m.isCommandMode() {
				m.actionOpen = !m.actionOpen
				m.actionCursor = 0
				if m.actionOpen {
					m.status = "Actions open."
				} else {
					m.status = "Actions closed."
				}
			}
			return m, nil
		case "up":
			if m.newProjectOpen {
				m.newProjectCursor--
				if m.newProjectCursor < 0 {
					m.newProjectCursor = 1
				}
				m.syncNewProjectFocus()
				return m, nil
			}
			if m.onboardingOpen {
				m.moveOnboarding(-1)
				return m, nil
			}
			if m.themePickerOpen {
				m.moveThemePicker(-1)
				return m, nil
			}
			if m.settingsOpen {
				m.settingsCursor--
				if m.settingsCursor < 0 {
					m.settingsCursor = m.settingsRowCount() - 1
				}
				return m, nil
			}
			if m.hiddenOpen {
				hidden := m.hiddenProjectPaths()
				if len(hidden) > 0 {
					m.hiddenCursor--
					if m.hiddenCursor < 0 {
						m.hiddenCursor = len(hidden) - 1
					}
				}
				return m, nil
			}
			if m.layoutOpen && !m.layoutDetailOpen && !m.layoutNaming {
				if m.layoutDeleteConfirm {
					return m, nil
				}
				m.moveLayoutSelection(-1)
				return m, nil
			}
			if m.layoutOpen && m.layoutDetailOpen && !m.layoutEditing {
				m.layoutMoveSelection("up")
				return m, nil
			}
			if m.viewFocus {
				// already at top, do nothing
			} else if m.isCommandMode() {
				entries := m.filteredCommands()
				if len(entries) == 0 {
					return m, nil
				}
				m.cmdCursor--
				if m.cmdCursor < 0 {
					m.cmdCursor = len(entries) - 1
				}
			} else if m.actionOpen {
				actions := m.projectActions()
				if len(actions) == 0 {
					return m, nil
				}
				m.actionCursor--
				if m.actionCursor < 0 {
					m.actionCursor = len(actions) - 1
				}
			} else {
				// enter view focus when already on first visible row
				cols := m.gridColumns()
				if m.displayMode == "list" {
					cols = 1
				}
				if m.cursor < cols && m.rowOffset == 0 {
					m.viewFocus = true
					modes := m.availableViewModes()
					for i, mode := range modes {
						if mode == m.currentViewLabel() {
							m.viewFocusCursor = i
							break
						}
					}
				} else {
					m.moveCursor(0, -1)
				}
			}
			return m, nil
		case "down":
			if m.newProjectOpen {
				m.newProjectCursor = (m.newProjectCursor + 1) % 2
				m.syncNewProjectFocus()
				return m, nil
			}
			if m.onboardingOpen {
				m.moveOnboarding(1)
				return m, nil
			}
			if m.themePickerOpen {
				m.moveThemePicker(1)
				return m, nil
			}
			if m.settingsOpen {
				m.settingsCursor++
				if m.settingsCursor >= m.settingsRowCount() {
					m.settingsCursor = 0
				}
				return m, nil
			}
			if m.hiddenOpen {
				hidden := m.hiddenProjectPaths()
				if len(hidden) > 0 {
					m.hiddenCursor++
					if m.hiddenCursor >= len(hidden) {
						m.hiddenCursor = 0
					}
				}
				return m, nil
			}
			if m.layoutOpen && !m.layoutDetailOpen && !m.layoutNaming {
				if m.layoutDeleteConfirm {
					return m, nil
				}
				m.moveLayoutSelection(1)
				return m, nil
			}
			if m.layoutOpen && m.layoutDetailOpen && !m.layoutEditing {
				m.layoutMoveSelection("down")
				return m, nil
			}
			if m.viewFocus {
				m.viewFocus = false
			} else if m.isCommandMode() {
				entries := m.filteredCommands()
				if len(entries) == 0 {
					return m, nil
				}
				m.cmdCursor++
				if m.cmdCursor >= len(entries) {
					m.cmdCursor = 0
				}
			} else if m.actionOpen {
				actions := m.projectActions()
				if len(actions) == 0 {
					return m, nil
				}
				m.actionCursor++
				if m.actionCursor >= len(actions) {
					m.actionCursor = 0
				}
			} else {
				m.moveCursor(0, 1)
			}
			return m, nil
		case "left":
			if m.onboardingOpen {
				m.adjustOnboarding(-1)
				return m, nil
			}
			if m.settingsOpen {
				m.adjustSetting(-1)
				return m, nil
			}
			if m.layoutOpen && m.layoutDetailOpen && !m.layoutEditing {
				m.layoutMoveSelection("left")
				return m, nil
			}
			if m.viewFocus {
				modes := m.availableViewModes()
				m.viewFocusCursor--
				if m.viewFocusCursor < 0 {
					m.viewFocusCursor = len(modes) - 1
				}
			} else if !m.isCommandMode() && !m.actionOpen && m.displayMode != "list" {
				m.moveCursor(-1, 0)
			}
			return m, nil
		case "right":
			if m.onboardingOpen {
				m.adjustOnboarding(1)
				return m, nil
			}
			if m.settingsOpen {
				m.adjustSetting(1)
				return m, nil
			}
			if m.layoutOpen && m.layoutDetailOpen && !m.layoutEditing {
				m.layoutMoveSelection("right")
				return m, nil
			}
			if m.viewFocus {
				modes := m.availableViewModes()
				m.viewFocusCursor++
				if m.viewFocusCursor >= len(modes) {
					m.viewFocusCursor = 0
				}
			} else if !m.isCommandMode() && !m.actionOpen && m.displayMode != "list" {
				m.moveCursor(1, 0)
			}
			return m, nil
		case "[":
			if !m.isCommandMode() && !m.actionOpen {
				m.cycleViewMode(-1)
			}
			return m, nil
		case "]":
			if !m.isCommandMode() && !m.actionOpen {
				m.cycleViewMode(1)
			}
			return m, nil
		case "f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8":
			if !m.isCommandMode() && !m.actionOpen {
				n := int(msg.String()[1] - '1') // "f1"->0, "f2"->1, ...
				modes := m.availableViewModes()
				if n < len(modes) {
					m.setViewMode(modes[n])
					m.viewFocus = false
				}
			}
			return m, nil
		case "enter":
			if m.onboardingOpen {
				if m.onboardingCursor == m.onboardingRowCount()-1 {
					return m, m.closeOnboarding()
				}
				m.adjustOnboarding(0)
				return m, nil
			}
			if m.newProjectOpen {
				return m, m.finishNewProjectImport()
			}
			if m.themePickerOpen {
				return m, m.closeThemePicker(true)
			}
			if m.settingsOpen {
				m.settingsOpen = false
				m.status = "Settings saved."
				return m, tea.Batch(
					saveStateCmd(m.stateStore, m.buildAppState(), ""),
					saveSettingsCmd(m.settingsStore, m.settings, "Settings saved."),
				)
			}
			if m.hiddenOpen {
				return m, m.restoreSelectedHiddenProject()
			}
			if m.layoutOpen {
				if m.layoutNaming {
					return m, m.finishNewLayout()
				}
				if m.layoutDeleteConfirm {
					if m.layoutName == "" {
						m.layoutDeleteConfirm = false
						m.status = "No layout selected."
						return m, nil
					}
					return m, deleteLayoutCmd(m.layoutProject, m.layoutName)
				}
				if m.layoutEditing {
					// confirm command edit
					if len(m.layoutPanes) > 0 {
						m.layoutPanes[m.layoutCursor].Command = m.layoutInput.Value()
						m.layoutDirty = true
						m.layoutSource = "edited"
					}
					m.layoutEditing = false
					m.layoutInput.Blur()
					return m, m.autosaveLayoutCmd()
				} else {
					if !m.layoutDetailOpen {
						if m.layoutName == "" {
							m.status = "Create a layout first."
							return m, nil
						}
						m.layoutDetailOpen = true
						m.status = fmt.Sprintf("Editing layout %s for %s.", m.layoutName, m.layoutProject.Name)
						return m, nil
					}
					// start editing command for selected pane
					m.layoutStartEdit()
				}
				return m, nil
			}
			if m.viewFocus {
				modes := m.availableViewModes()
				if m.viewFocusCursor >= 0 && m.viewFocusCursor < len(modes) {
					m.setViewMode(modes[m.viewFocusCursor])
				}
				m.viewFocus = false
				return m, nil
			}
			return m, m.handleEnter()
		}
	}

	if m.loadingProjects {
		return m, nil
	}

	if keyMsg, isKey := msg.(tea.KeyMsg); isKey {
		if m.newProjectOpen {
			var cmd tea.Cmd
			if m.newProjectCursor == 0 {
				m.newProjectPathInput, cmd = m.newProjectPathInput.Update(msg)
				m.newProjectPathInput.SetValue(stripLeakedTerminalFragments(m.newProjectPathInput.Value()))
			} else {
				m.newProjectNameInput, cmd = m.newProjectNameInput.Update(msg)
				m.newProjectNameInput.SetValue(stripLeakedTerminalFragments(m.newProjectNameInput.Value()))
			}
			return m, cmd
		}

		// Layout editor single-key actions (when panel open, not editing a field)
		if m.layoutOpen && !m.layoutEditing {
			if m.layoutNaming {
				var cmd tea.Cmd
				m.layoutNameInput, cmd = m.layoutNameInput.Update(msg)
				return m, cmd
			}
			switch keyMsg.String() {
			case "e":
				if !m.layoutDetailOpen {
					if m.layoutName == "" {
						m.status = "Create a layout first."
						return m, nil
					}
					m.layoutDetailOpen = true
					m.status = fmt.Sprintf("Editing layout %s for %s.", m.layoutName, m.layoutProject.Name)
					return m, nil
				}
				m.layoutStartEdit()
				return m, nil
			case "a", "r":
				if !m.layoutDetailOpen {
					m.layoutDetailOpen = true
				}
				m.layoutSplitSelected("right")
				return m, m.autosaveLayoutCmd()
			case "b":
				if !m.layoutDetailOpen {
					m.layoutDetailOpen = true
				}
				m.layoutSplitSelected("down")
				return m, m.autosaveLayoutCmd()
			case "d":
				if !m.layoutDetailOpen {
					return m, nil
				}
				m.layoutDeletePane()
				return m, m.autosaveLayoutCmd()
			case "t":
				if !m.layoutDetailOpen {
					return m, nil
				}
				m.layoutToggleDirection()
				return m, m.autosaveLayoutCmd()
			case "l":
				m.layoutLoadSaved()
				return m, nil
			case "n":
				m.beginNewLayout()
				return m, nil
			case "f":
				if m.layoutName == "" {
					m.status = "Create a layout first."
					return m, nil
				}
				return m, setDefaultLayoutCmd(m.layoutProject, m.layoutName)
			case "x":
				if m.layoutDetailOpen || m.layoutNaming || m.layoutName == "" {
					return m, nil
				}
				m.layoutDeleteConfirm = true
				m.status = fmt.Sprintf("Delete layout %s? Enter to confirm, Esc to cancel.", m.layoutName)
				return m, nil
			case "g":
				return m, grabLayoutFromGhosttyCmd()
			case "p":
				return m, applyLayoutCmd(m.layoutProject, m.layoutPanes)
			case "s":
				if m.layoutName == "" {
					m.beginNewLayout()
					return m, nil
				}
				return m, saveLayoutCmd(m.layoutProject, m.layoutName, m.layoutPanes, false)
			}
			return m, nil
		}

		// Route keys to layoutInput when editing a pane command
		if m.layoutOpen && m.layoutEditing {
			var cmd tea.Cmd
			m.layoutInput, cmd = m.layoutInput.Update(msg)
			return m, cmd
		}
	}

	if m.onboardingOpen || m.newProjectOpen || m.themePickerOpen || m.settingsOpen || m.hiddenOpen || m.actionOpen {
		return m, nil
	}

	var cmd tea.Cmd
	prev := m.input.Value()
	m.input, cmd = m.input.Update(msg)
	m.input.SetValue(stripLeakedTerminalFragments(m.input.Value()))
	if m.input.Value() != prev {
		m.cmdCursor = 0
		if !m.isCommandMode() {
			m.cursor = 0
			m.rowOffset = 0
		}
	}
	m.syncCursor()
	return m, cmd
}

func (m model) View() string {
	if m.onboardingOpen && !m.loadingProjects {
		return renderOnboardingScreen(m)
	}

	showHeader, showViewBar, showFooter, showHints := m.chromeVisibility()
	contentWidth := m.contentWidth()
	commandBar := fitOuterWidth(m.inputShellStyle(), contentWidth).Render(m.input.View())
	content := renderContent(m, contentWidth)
	actionPanel := renderActionPanel(m, contentWidth)
	if actionPanel != "" && m.overlayActionPanel() {
		content = overlayTop(content, clampLines(actionPanel, m.overlayLineBudget()))
		actionPanel = ""
	}

	parts := make([]string, 0, 12)
	appendPart := func(value string) {
		if value == "" {
			return
		}
		if len(parts) > 0 {
			parts = append(parts, "")
		}
		parts = append(parts, value)
	}

	if showHeader {
		header := currentStyles.header.Render("repodock") + "  " + currentStyles.headerVersion.Render(buildinfo.HeaderVersion())
		appendPart(header)
	}

	appendPart(commandBar)

	if showViewBar {
		appendPart(renderViewBar(m, contentWidth))
	}

	appendPart(content)

	if actionPanel != "" {
		appendPart(actionPanel)
	}

	if showFooter {
		appendPart(renderSingleLine("", m.status, contentWidth, currentStyles.status))
	}

	if showHints {
		var hintText string
		if m.loadingProjects {
			hintText = "loading providers  ·  building project list"
		} else if m.onboardingOpen {
			hintText = "enter close  ·  esc close  ·  /onboard reopen later"
		} else if m.newProjectOpen {
			hintText = "paste path or drag folder into path box  ·  tab/↑↓ switch field  ·  enter import  ·  esc close"
		} else if m.themePickerOpen {
			hintText = "↑↓ preview  ·  enter apply  ·  esc cancel  ·  q close"
		} else if m.settingsOpen {
			hintText = "↑↓ navigate  ·  ←→ change  ·  esc/enter close"
		} else if m.hiddenOpen {
			hintText = "↑↓ select  ·  enter restore  ·  esc close  ·  /unhide-all restore all"
		} else if m.layoutOpen {
			if m.layoutNaming {
				hintText = "type name  ·  enter create  ·  esc cancel"
			} else if m.layoutEditing {
				hintText = "enter confirm  ·  esc cancel"
			} else if m.layoutDetailOpen {
				hintText = "↑↓ select pane  ·  ← parent  ·  → child  ·  r split right  ·  b split down  ·  t toggle  ·  e edit cmd  ·  d del  ·  s save  ·  p apply  ·  esc back"
			} else if m.layoutDeleteConfirm {
				hintText = "enter confirm delete  ·  esc cancel"
			} else {
				hintText = "↑↓ select layout  ·  enter edit  ·  n new  ·  x delete  ·  l load  ·  g grab  ·  s save  ·  p apply  ·  esc close"
			}
		} else if m.viewFocus {
			hintText = "←→ switch view  ·  enter confirm  ·  ↓/esc back to grid  ·  q quit"
		} else if m.isCommandMode() {
			hintText = "↑↓ navigate  ·  enter run  ·  esc cancel  ·  q quit"
		} else if m.actionOpen {
			hintText = "tab actions  ·  ↑↓ navigate  ·  enter run  ·  esc close  ·  q quit"
		} else {
			hintText = "search  ·  ↑ to view bar  ·  [ ] switch view  ·  wheel scroll  ·  tab actions  ·  enter shell  ·  q quit"
		}
		appendPart(renderSingleLine("", hintText, contentWidth, currentStyles.subtle))
	}

	page := currentStyles.page.Copy().Padding(m.pagePaddingY(), m.pagePaddingX())
	return clampLines(page.Render(lipgloss.JoinVertical(lipgloss.Left, parts...)), m.height)
}

func (m *model) handleEnter() tea.Cmd {
	value := strings.TrimSpace(m.input.Value())
	if strings.HasPrefix(value, "/") {
		entries := m.filteredCommands()
		name := strings.TrimPrefix(value, "/")
		if len(entries) > 0 && m.cmdCursor < len(entries) {
			name = entries[m.cmdCursor].name
		}
		cmd := m.runCommand(name)
		m.input.SetValue("")
		m.cmdCursor = 0
		m.syncCursor()
		return cmd
	}

	if m.actionOpen {
		return m.runSelectedAction()
	}

	project, ok := m.selectedProject()
	if !ok {
		m.status = "No project selected."
		return nil
	}

	m.status = fmt.Sprintf("Opening shell in %s...", project.Name)
	return m.openProjectShell(project)
}

func (m *model) runCommand(raw string) tea.Cmd {
	command := strings.ToLower(strings.TrimSpace(raw))
	if strings.HasPrefix(command, "view ") {
		mode := strings.TrimSpace(strings.TrimPrefix(command, "view "))
		if m.setViewMode(mode) {
			return nil
		}
		m.status = fmt.Sprintf("Unknown view mode: %s", mode)
		return nil
	}
	if strings.HasPrefix(command, "theme ") {
		family := strings.TrimSpace(strings.TrimPrefix(command, "theme "))
		return m.setThemeFamily(family)
	}
	if strings.HasPrefix(command, "mode ") {
		mode := strings.TrimSpace(strings.TrimPrefix(command, "mode "))
		return m.setThemeMode(mode)
	}
	if strings.HasPrefix(command, "palette ") {
		palette := strings.TrimSpace(strings.TrimPrefix(command, "palette "))
		return m.setDataPalette(palette)
	}
	if strings.HasPrefix(command, "demo ") {
		switch strings.TrimSpace(strings.TrimPrefix(command, "demo ")) {
		case "on":
			return m.setDemoMode(true)
		case "off":
			return m.setDemoMode(false)
		}
	}
	if strings.HasPrefix(command, "reveal ") {
		mode := strings.TrimSpace(strings.TrimPrefix(command, "reveal "))
		project, ok := m.selectedProject()
		if !ok {
			m.status = "No project selected."
			return nil
		}
		switch mode {
		case "reveal", "open":
			m.status = fmt.Sprintf("Opening %s in Finder (%s)...", project.Name, mode)
			return revealInFinderCmd(project.Path, mode)
		}
	}
	if strings.HasPrefix(command, "new ") {
		path := strings.TrimSpace(strings.TrimPrefix(command, "new "))
		return m.addManualProject(path)
	}
	if strings.HasPrefix(command, "rename ") {
		name := strings.TrimSpace(strings.TrimPrefix(command, "rename "))
		return m.renameSelectedProject(name)
	}

	switch command {
	case "":
		m.status = "Command bar cleared."
		return nil
	case "help":
		m.status = "Commands: /help /demo [/demo on|off] /onboard /providers /view mixed|provider /layout /theme [/theme family] /mode auto|dark|light /palette tableau10 /sync /shell /reveal [/reveal reveal|open] /new [/new /absolute/path] /rename New Name /pin /unpin /hide /hidden /unhide-all /copy /clear"
		return nil
	case "demo":
		return m.setDemoMode(!m.demoMode())
	case "onboard":
		m.openOnboarding()
		return nil
	case "providers":
		m.openProviders()
		return nil
	case "view":
		m.status = fmt.Sprintf("Current view: %s.", m.currentViewLabel())
		return nil
	case "layout":
		project, ok := m.selectedProject()
		if !ok {
			m.status = "No project selected."
			return nil
		}
		m.openLayoutEditor(project)
		m.status = fmt.Sprintf("Editing layout for %s.", project.Name)
		return nil
	case "theme":
		m.openThemePicker()
		return nil
	case "mode":
		m.status = fmt.Sprintf("Theme mode: %s.", m.resolvedTheme.Mode)
		return nil
	case "palette":
		m.status = fmt.Sprintf("Data palette: %s.", activeTheme.Palette.Data.Name)
		return nil
	case "grid":
		m.displayMode = "grid"
		m.syncCursor()
		m.status = "Switched to grid display."
		return saveStateCmd(m.stateStore, m.buildAppState(), "")
	case "list":
		m.displayMode = "list"
		m.syncCursor()
		m.status = "Switched to list display."
		return saveStateCmd(m.stateStore, m.buildAppState(), "")
	case "settings":
		m.settingsOpen = true
		m.settingsCursor = 0
		m.status = "Settings open."
		return nil
	case "hidden":
		m.openHiddenProjects()
		return nil
	case "new":
		m.openNewProjectImport()
		return nil
	case "sync":
		m.status = "Reloading project providers..."
		m.loadingProjects = true
		m.loadingProjectsFrame = 0
		return tea.Batch(loadProjectsCmd(m.stateStore), projectLaunchTickCmd())
	case "shell":
		project, ok := m.selectedProject()
		if !ok {
			m.status = "No project selected."
			return nil
		}
		m.status = fmt.Sprintf("Opening shell in %s...", project.Name)
		return m.openProjectShell(project)
	case "reveal":
		project, ok := m.selectedProject()
		if !ok {
			m.status = "No project selected."
			return nil
		}
		mode := m.finderRevealMode()
		m.status = fmt.Sprintf("Opening %s in Finder (%s)...", project.Name, mode)
		return revealInFinderCmd(project.Path, mode)
	case "default-layout":
		project, ok := m.selectedProject()
		if !ok {
			m.status = "No project selected."
			return nil
		}
		layout, err := store.NewLayoutStore(project.Path).Load()
		if err != nil {
			m.status = fmt.Sprintf("Failed to load default layout: %v", err)
			return nil
		}
		if len(layout.Panes) == 0 {
			m.status = fmt.Sprintf("No saved default layout for %s.", project.Name)
			return nil
		}
		m.status = fmt.Sprintf("Loading default layout for %s...", project.Name)
		return applyLayoutCmd(project, layout.Panes)
	case "finder":
		project, ok := m.selectedProject()
		if !ok {
			m.status = "No project selected."
			return nil
		}
		mode := m.finderRevealMode()
		m.status = fmt.Sprintf("Opening %s in Finder (%s)...", project.Name, mode)
		return revealInFinderCmd(project.Path, mode)
	case "pin":
		return m.setPinnedForSelected(true)
	case "unpin":
		return m.setPinnedForSelected(false)
	case "hide":
		return m.setHiddenForSelected(true)
	case "unhide-all":
		return m.clearHiddenProjects()
	case "copy":
		project, ok := m.selectedProject()
		if !ok {
			m.status = "No project selected."
			return nil
		}
		m.status = fmt.Sprintf("Copying path for %s...", project.Name)
		return copyPathCmd(project.Path)
	case "codex":
		m.status = "Codex launch is not wired yet."
		return nil
	case "claude":
		m.status = "Claude launch is not wired yet."
		return nil
	case "code":
		m.status = "VS Code launch is not wired yet."
		return nil
	case "cursor":
		m.status = "Cursor launch is not wired yet."
		return nil
	case "clear":
		m.status = "Status cleared."
		return nil
	default:
		m.status = fmt.Sprintf("Unknown command: /%s", command)
		return nil
	}
}

func (m *model) moveCursor(dx, dy int) {
	projects := m.filteredProjects()
	if len(projects) == 0 {
		m.cursor = 0
		return
	}

	columns := m.gridColumns()
	if m.displayMode == "list" {
		columns = 1
		dx = 0 // no horizontal movement in list mode
	}
	row := m.cursor / columns
	col := m.cursor % columns

	col += dx
	row += dy

	if col < 0 {
		col = 0
	}
	if row < 0 {
		row = 0
	}
	if col >= columns {
		col = columns - 1
	}

	next := row*columns + col
	if next >= len(projects) {
		last := len(projects) - 1
		row = last / columns
		if row*columns+col > last {
			col = last % columns
		}
		next = row*columns + col
	}

	if next < 0 {
		next = 0
	}
	if next >= len(projects) {
		next = len(projects) - 1
	}

	m.cursor = next
	m.syncViewport()
	m.status = fmt.Sprintf("Selected %s", projects[m.cursor].Name)
}

func (m *model) syncCursor() {
	projects := m.filteredProjects()
	if len(projects) == 0 {
		m.cursor = 0
		return
	}

	if m.cursor >= len(projects) {
		m.cursor = len(projects) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.syncViewport()
}

func (m *model) syncInputWidth() {
	width := m.contentWidth() - m.inputShellStyle().GetHorizontalFrameSize()
	if width < 1 {
		width = 1
	}
	m.input.Width = width
}

func (m *model) syncViewport() {
	projects := m.filteredProjects()
	columns := m.gridColumns()
	if m.displayMode == "list" {
		columns = 1
	}
	maxRows := m.visibleRows()
	if len(projects) == 0 || columns <= 0 || maxRows <= 0 {
		m.rowOffset = 0
		return
	}

	totalRows := (len(projects) + columns - 1) / columns
	maxOffset := max(totalRows-maxRows, 0)
	if m.rowOffset > maxOffset {
		m.rowOffset = maxOffset
	}
	if m.rowOffset < 0 {
		m.rowOffset = 0
	}

	selectedRow := m.cursor / columns
	if selectedRow < m.rowOffset {
		m.rowOffset = selectedRow
	}
	if selectedRow >= m.rowOffset+maxRows {
		m.rowOffset = selectedRow - maxRows + 1
	}
	if m.rowOffset > maxOffset {
		m.rowOffset = maxOffset
	}
}

func (m *model) scrollViewport(delta int) {
	projects := m.filteredProjects()
	columns := m.gridColumns()
	if m.displayMode == "list" {
		columns = 1
	}
	maxRows := m.visibleRows()
	if len(projects) == 0 || columns <= 0 || maxRows <= 0 {
		return
	}

	totalRows := (len(projects) + columns - 1) / columns
	maxOffset := max(totalRows-maxRows, 0)
	m.rowOffset += delta
	if m.rowOffset < 0 {
		m.rowOffset = 0
	}
	if m.rowOffset > maxOffset {
		m.rowOffset = maxOffset
	}
}

func (m model) filteredProjects() []domain.Project {
	projects := m.projects
	if m.viewMode != "" && m.viewMode != "mixed" {
		filteredByView := make([]domain.Project, 0, len(projects))
		for _, project := range projects {
			if projectHasSource(project, m.viewMode) {
				filteredByView = append(filteredByView, project)
			}
		}
		projects = filteredByView
	}

	query := strings.TrimSpace(strings.ToLower(m.input.Value()))
	if query == "" || strings.HasPrefix(query, "/") {
		return projects
	}

	filtered := make([]domain.Project, 0, len(projects))
	for _, project := range projects {
		haystack := strings.ToLower(project.Name + " " + project.Path + " " + strings.Join(sourceNames(project.Sources), " "))
		if strings.Contains(haystack, query) {
			filtered = append(filtered, project)
		}
	}
	return filtered
}

func (m model) isCommandMode() bool {
	return strings.HasPrefix(strings.TrimSpace(m.input.Value()), "/")
}

func (m model) filteredCommands() []commandEntry {
	query := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(m.input.Value(), "/")))
	if query == "" {
		return allCommands
	}
	out := make([]commandEntry, 0, len(allCommands))
	for _, e := range allCommands {
		if strings.Contains(e.name, query) || strings.Contains(e.desc, query) {
			out = append(out, e)
		}
	}
	return out
}

func (m model) gridColumns() int {
	if m.width == 0 {
		return 1
	}
	usable := m.contentWidth()
	minOuterWidth := m.minCardOuterWidth()
	cols := (usable + 1) / (minOuterWidth + 1)
	if cols < 1 {
		return 1
	}
	for cols > 1 {
		cellOuterWidth := (usable - (cols - 1)) / cols
		if cellOuterWidth >= minOuterWidth {
			break
		}
		cols--
	}
	return cols
}

func (m model) pagePaddingX() int {
	switch {
	case m.width < 72:
		return 0
	case m.width < 110:
		return 1
	default:
		return 2
	}
}

func (m model) pagePaddingY() int {
	if m.height < 16 {
		return 0
	}
	return 1
}

func (m model) contentWidth() int {
	width := m.width - (m.pagePaddingX() * 2)
	if width < 1 {
		return 1
	}
	return width
}

// These are the coarse eligibility gates. The final visibility decision is
// resolved by chromeVisibility so the content area always wins when height is
// tight.
func (m model) headerEligible() bool  { return m.height >= 9 && m.contentWidth() >= 24 }
func (m model) viewBarEligible() bool { return m.height >= 12 && m.contentWidth() >= 28 }
func (m model) hintsEligible() bool   { return m.height >= 16 && m.contentWidth() >= 52 }
func (m model) footerEligible() bool  { return m.height >= 7 }
func (m model) overlayActionPanel() bool {
	return true
}

func (m model) shouldDockActionPanelSide() bool {
	return false
}

func (m model) actionPanelDockWidth() int {
	return 0
}

func (m model) overlayLineBudget() int {
	budget := m.height - m.chromeHeight()
	if budget < 3 {
		return 3
	}
	return budget
}

func (m model) commandBarOuterHeight() int {
	return 1 + m.inputShellStyle().GetVerticalFrameSize()
}

func (m model) inputShellStyle() lipgloss.Style {
	style := currentStyles.inputShell
	if m.contentWidth() < 72 {
		style = style.Padding(0, 0)
	}
	return style
}

func (m model) cardStyle() lipgloss.Style {
	style := currentStyles.card
	if m.contentWidth() < 120 {
		style = style.Padding(0, 1)
	}
	return style
}

func (m model) selectedCardStyle() lipgloss.Style {
	style := currentStyles.selectedCard
	if m.contentWidth() < 120 {
		style = style.Padding(0, 1)
	}
	return style
}

func (m model) minCardOuterWidth() int {
	frame := m.cardStyle().GetHorizontalFrameSize()
	minimum := frame + 8
	var target int
	switch {
	case m.contentWidth() < 72:
		switch m.gridWidthMode() {
		case "compact":
			target = 18
		case "wide":
			target = 30
		default:
			target = 22
		}
	case m.contentWidth() < 110:
		switch m.gridWidthMode() {
		case "compact":
			target = 24
		case "wide":
			target = 36
		default:
			target = 28
		}
	default:
		switch m.gridWidthMode() {
		case "compact":
			target = 30
		case "wide":
			target = 48
		default:
			target = 36
		}
	}

	return max(minimum, target)
}

// chipPlacement holds the pre-rendered chip string and its position.
// All layout consumers (height, render, hit-test) share this struct so they
// can never disagree about where a chip lands.
type chipPlacement struct {
	mode     string
	rendered string // final ANSI string, ready to print
	row      int    // 0-based row within the view bar
	x        int    // column offset within that row (excl. pagePaddingX)
	width    int    // terminal cell width of rendered
}

// chipPlacements renders every chip with its correct visual style and computes
// (row, x) positions using the same wrapping algorithm as the actual render.
// This is the single source of truth for viewBarHeight, renderViewBar, and
// viewModeAt -- all three are guaranteed to agree on the layout.
func (m model) chipPlacements(width int) []chipPlacement {
	modes := m.availableViewModes()
	placements := make([]chipPlacement, 0, len(modes))
	curX, curRow := 0, 0

	for i, mode := range modes {
		label := m.viewChipLabel(mode)
		sourceColor := dataColorForViewMode(mode)
		isActive := mode == m.currentViewLabel()
		isFocused := m.viewFocus && i == m.viewFocusCursor
		isHovered := m.hoverViewMode == mode

		var style lipgloss.Style
		switch {
		case isFocused || isHovered:
			style = currentStyles.viewChip.
				Bold(true).
				Foreground(lipgloss.Color("#000000")).
				Background(lipgloss.Color("#FFFFFF")).
				BorderForeground(lipgloss.Color("#FFFFFF"))
		case isActive:
			style = currentStyles.activeViewChip.Bold(true)
			if sourceColor != "" {
				style = style.Foreground(lipgloss.Color(sourceColor)).
					BorderForeground(lipgloss.Color(sourceColor))
			}
		default:
			style = currentStyles.viewChip
			if sourceColor != "" {
				style = style.Foreground(lipgloss.Color(sourceColor)).
					BorderForeground(lipgloss.Color(sourceColor))
			}
		}

		chip := style.Render(label)
		chipW := lipgloss.Width(chip)
		spacer := 0
		if i > 0 {
			spacer = 1
		}
		if curX+spacer+chipW > width && curX > 0 {
			curRow++
			curX = 0
			spacer = 0
		}
		placements = append(placements, chipPlacement{
			mode: mode, rendered: chip,
			row: curRow, x: curX + spacer, width: chipW,
		})
		curX += spacer + chipW
	}
	return placements
}

// viewBarHeight returns how many terminal rows the view bar chips occupy.
func (m model) viewBarHeightFor(showViewBar bool) int {
	if !showViewBar {
		return 0
	}
	pl := m.chipPlacements(m.contentWidth())
	if len(pl) == 0 {
		return 1
	}
	return pl[len(pl)-1].row + 1
}

func (m model) viewBarHeight() int {
	_, showViewBar, _, _ := m.chromeVisibility()
	return m.viewBarHeightFor(showViewBar)
}

// chromeHeight returns the number of terminal rows consumed by fixed UI
// elements given the current height.
func (m model) chromeHeight() int {
	showHeader, showViewBar, showFooter, showHints := m.chromeVisibility()
	return m.chromeHeightFor(showHeader, showViewBar, showFooter, showHints)
}

func (m model) chromeHeightFor(showHeader, showViewBar, showFooter, showHints bool) int {
	h := m.pagePaddingY()*2 + m.commandBarOuterHeight()
	sections := 2 // commandBar + content each create a section boundary
	if showHeader {
		h += 1
		sections++
	}
	if showViewBar {
		h += m.viewBarHeightFor(showViewBar)
		sections++
	}
	if showFooter {
		h += 1
		sections++
	}
	if showHints {
		h += 1
		sections++
	}
	h += max(0, sections-1) * sectionGapHeight
	return h
}

func (m model) minContentHeight() int {
	if m.displayMode == "list" {
		return 1
	}
	return cardHeight + m.cardStyle().GetVerticalFrameSize()
}

func (m model) contentAreaHeightFor(showFooter, showHints bool) int {
	usable := m.height - m.chromeHeightFor(m.headerEligible(), m.viewBarEligible(), showFooter, showHints)
	if usable < 0 {
		return 0
	}
	return usable
}

func (m model) contentAreaHeightForAll(showHeader, showViewBar, showFooter, showHints bool) int {
	usable := m.height - m.chromeHeightFor(showHeader, showViewBar, showFooter, showHints)
	if usable < 0 {
		return 0
	}
	return usable
}

func (m model) chromeVisibility() (showHeader, showViewBar, showFooter, showHints bool) {
	showHeader = m.headerEligible()
	showViewBar = m.viewBarEligible()
	showFooter = m.footerEligible()
	showHints = m.hintsEligible()

	minContent := m.minContentHeight()
	if m.contentAreaHeightForAll(showHeader, showViewBar, showFooter, showHints) >= minContent {
		return showHeader, showViewBar, showFooter, showHints
	}

	if showHints && m.contentAreaHeightForAll(showHeader, showViewBar, showFooter, false) >= minContent {
		return showHeader, showViewBar, showFooter, false
	}

	if showHints {
		showHints = false
	}
	if showFooter && m.contentAreaHeightForAll(showHeader, showViewBar, showFooter, showHints) < minContent {
		showFooter = false
	}
	if showViewBar && m.contentAreaHeightForAll(showHeader, showViewBar, showFooter, showHints) < minContent {
		showViewBar = false
	}
	if showHeader && m.contentAreaHeightForAll(showHeader, showViewBar, showFooter, showHints) < minContent {
		showHeader = false
	}

	if m.contentAreaHeightForAll(showHeader, showViewBar, showFooter, showHints) < minContent {
		showHints = false
		showFooter = false
	}

	return showHeader, showViewBar, showFooter, showHints
}

func (m model) gridRows() int {
	usable := m.height - m.chromeHeight()
	cardTotalHeight := cardHeight + m.cardStyle().GetVerticalFrameSize()
	rows := usable / cardTotalHeight
	if rows < 1 {
		return 1
	}
	return rows
}

func (m model) listRows() int {
	// Keep one extra line of safety in list mode. The list is much more likely
	// than the grid to hit a "just barely fits" terminal height and push the
	// footer/hints off-screen.
	usable := m.height - m.chromeHeight() - 1
	if usable < 1 {
		return 1
	}
	return usable
}

func (m model) visibleRows() int {
	if m.displayMode == "list" {
		return m.listRows()
	}
	return m.gridRows()
}

func renderGrid(m model, width int) string {
	projects := m.filteredProjects()
	width = max(1, width)
	cardStyle := m.cardStyle()
	selectedCardStyle := m.selectedCardStyle()

	if len(projects) == 0 {
		return currentStyles.empty.
			Width(width).
			Padding(1, 0).
			Align(lipgloss.Center).
			Render("No projects match.")
	}

	columns := m.gridColumns()
	maxRows := m.gridRows()
	cardTotalHeight := cardHeight + cardStyle.GetVerticalFrameSize()
	columnWidths := gridColumnWidths(width, columns)

	// spacer between columns: 1 char wide, same height as a rendered card
	spacerLines := make([]string, cardTotalHeight)
	for i := range spacerLines {
		spacerLines[i] = " "
	}
	colSpacer := strings.Join(spacerLines, "\n")

	rowCount := 0
	rows := make([]string, 0, (len(projects)+columns-1)/columns)
	for row := m.rowOffset; row*columns < len(projects); row++ {
		if rowCount >= maxRows {
			break
		}
		rowCount++
		start := row * columns
		cells := make([]string, 0, columns*2-1)
		for offset := 0; offset < columns; offset++ {
			if offset > 0 {
				cells = append(cells, colSpacer)
			}
			cellOuterWidth := columnWidths[offset]
			index := start + offset
			if index >= len(projects) {
				// invisible placeholder, same outer dimensions as a card
				empty := lipgloss.NewStyle().
					Width(cellOuterWidth).
					Height(cardTotalHeight).
					Render("")
				cells = append(cells, empty)
				continue
			}

			project := projects[index]
			var lo time.Time
			if m.showLastOpened && m.lastOpened != nil {
				lo = m.lastOpened[filepath.Clean(project.Path)]
			}
			inGhostty := m.settings.Ghostty.Indicator && m.ghosttyOpen != nil
			if inGhostty {
				_, inGhostty = m.ghosttyOpen[filepath.Clean(project.Path)]
			}
			card := renderProjectCard(project, cellOuterWidth, index == m.cursor, m.isPinned(project.Path), lo, inGhostty, cardStyle, selectedCardStyle)
			cells = append(cells, card)
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func renderContent(m model, width int) string {
	if m.loadingProjects {
		return renderLoadingProjectsPanel(m, width)
	}
	var base string
	if m.displayMode == "list" {
		base = renderList(m, width)
	} else {
		base = renderGrid(m, width)
	}
	if m.themePickerOpen {
		return overlayTop(base, clampLines(renderThemePickerPanel(m, width), m.overlayLineBudget()))
	}
	if m.newProjectOpen {
		return overlayTop(base, clampLines(renderNewProjectPanel(m, width, m.overlayLineBudget()), m.overlayLineBudget()))
	}
	if m.settingsOpen {
		return overlayTop(base, renderSettingsPanel(m, width, m.overlayLineBudget()))
	}
	if m.hiddenOpen {
		return overlayTop(base, renderHiddenPanel(m, width, m.overlayLineBudget()))
	}
	if m.layoutOpen {
		return overlayTop(base, clampLines(renderLayoutPanel(m, width), m.overlayLineBudget()))
	}
	if !m.isCommandMode() {
		return base
	}
	return overlayTop(base, renderCommandPanel(m, width, m.overlayLineBudget()))
}

func renderViewBar(m model, width int) string {
	placements := m.chipPlacements(width)
	if len(placements) == 0 {
		return ""
	}

	// Group placements by row
	numRows := placements[len(placements)-1].row + 1
	rowChips := make([][]string, numRows)
	for _, p := range placements {
		rowChips[p.row] = append(rowChips[p.row], p.rendered)
	}

	rows := make([]string, 0, numRows)
	for _, chips := range rowChips {
		// Insert 1-space gaps between chips in the same row
		parts := make([]string, 0, len(chips)*2-1)
		for i, chip := range chips {
			if i > 0 {
				parts = append(parts, " ")
			}
			parts = append(parts, chip)
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, parts...))
	}

	return lipgloss.NewStyle().Width(width).Render(strings.Join(rows, "\n"))
}

func renderList(m model, width int) string {
	projects := m.filteredProjects()
	if len(projects) == 0 {
		return currentStyles.empty.
			Width(width).
			Padding(1, 0).
			Align(lipgloss.Center).
			Render("No projects match.")
	}

	width = max(1, width)
	maxRows := m.listRows()
	showTime := m.showLastOpened && width >= 72
	showSource := width >= 40
	showPath := width >= 56
	timeWidth := 0
	if showTime {
		timeWidth = 10
	}
	separatorCount := 0
	if showPath {
		separatorCount++
	}
	if showSource {
		separatorCount++
	}
	if showTime {
		separatorCount++
	}
	available := width - 2 - (separatorCount * 2)
	if available < 1 {
		available = 1
	}
	if showTime {
		available -= timeWidth
	}
	sourceWidth := 0
	if showSource {
		sourceWidth = min(18, max(10, available/4))
		available -= sourceWidth
	}
	nameWidth := min(24, max(12, available/3))
	if !showPath {
		nameWidth = max(1, available)
	}
	if nameWidth > available {
		nameWidth = available
	}
	pathWidth := 0
	if showPath {
		pathWidth = available - nameWidth
		if pathWidth < 12 {
			showPath = false
			pathWidth = 0
			nameWidth = max(1, available)
		}
	}

	lines := make([]string, 0, min(len(projects), maxRows))
	end := m.rowOffset + maxRows
	if end > len(projects) {
		end = len(projects)
	}
	for i := m.rowOffset; i < end; i++ {
		p := projects[i]
		selected := i == m.cursor
		pinned := m.isPinned(p.Path)
		openInGhostty := m.settings.Ghostty.Indicator && m.ghosttyOpen != nil
		if openInGhostty {
			_, openInGhostty = m.ghosttyOpen[filepath.Clean(p.Path)]
		}

		srcLabel := strings.Join(sourceNames(p.Sources), " · ")
		if pinned {
			srcLabel = "pin · " + srcLabel
		}

		var indicator, name, path, src string
		if selected {
			indicator = currentStyles.listIndicator.Render("▶ ")
			name = currentStyles.listSelected.Copy().Bold(true).Render(truncate(p.Name, nameWidth))
			if showPath {
				path = currentStyles.listSelectedAlt.Render(truncate(p.Path, pathWidth))
			}
			if showSource {
				src = lipgloss.NewStyle().Foreground(lipgloss.Color(primarySourceColor(p.Sources))).Render(truncate(srcLabel, sourceWidth))
			}
		} else {
			indicator = "  "
			name = currentStyles.listNormal.Render(truncate(p.Name, nameWidth))
			if showPath {
				path = currentStyles.listNormalAlt.Render(truncate(p.Path, pathWidth))
			}
			if showSource {
				src = lipgloss.NewStyle().Foreground(lipgloss.Color(primarySourceColor(p.Sources))).Render(truncate(srcLabel, sourceWidth))
			}
		}

		if openInGhostty && nameWidth >= 4 {
			name = truncate(strings.TrimSpace(p.Name)+" ●", nameWidth)
			if selected {
				name = currentStyles.listSelected.Copy().Bold(true).Render(name)
			} else {
				name = currentStyles.listNormal.Render(name)
			}
		}

		parts := []string{indicator + lipgloss.NewStyle().Width(nameWidth).Render(name)}
		if showPath {
			parts = append(parts, "  ")
			parts = append(parts, lipgloss.NewStyle().Width(pathWidth).Render(path))
		}
		if showSource {
			parts = append(parts, "  ")
			parts = append(parts, lipgloss.NewStyle().Width(sourceWidth).Render(src))
		}
		if showTime {
			var lo time.Time
			if m.lastOpened != nil {
				lo = m.lastOpened[filepath.Clean(p.Path)]
			}
			timeStr := relativeTime(lo)
			timeStyle := currentStyles.subtle
			if selected {
				timeStyle = currentStyles.listSelectedAlt
			}
			parts = append(parts, "  ")
			parts = append(parts, lipgloss.NewStyle().Width(timeWidth).Render(timeStyle.Render(timeStr)))
		}
		row := lipgloss.JoinHorizontal(lipgloss.Left, parts...)
		lines = append(lines, row)
	}

	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

func renderProjectCard(project domain.Project, outerWidth int, selected bool, pinned bool, lastOpenedAt time.Time, openInGhostty bool, normalStyle lipgloss.Style, activeStyle lipgloss.Style) string {
	sourceLabel := strings.Join(sourceNames(project.Sources), " · ")
	if pinned {
		sourceLabel = "pin · " + sourceLabel
	}

	srcColor := lipgloss.NewStyle().Foreground(lipgloss.Color(primarySourceColor(project.Sources)))
	var sourceLine string
	innerWidth := max(1, outerWidth-normalStyle.GetHorizontalFrameSize())
	if !lastOpenedAt.IsZero() {
		timeStr := relativeTime(lastOpenedAt)
		srcPart := truncate(sourceLabel, innerWidth-lipgloss.Width(timeStr)-2)
		padding := innerWidth - lipgloss.Width(srcPart) - lipgloss.Width(timeStr)
		if padding < 1 {
			padding = 1
		}
		sourceLine = srcColor.Render(srcPart) + strings.Repeat(" ", padding) + currentStyles.subtle.Render(timeStr)
	} else {
		sourceLine = srcColor.Render(truncate(sourceLabel, innerWidth))
	}

	nameLine := currentStyles.projectName.Render(truncate(project.Name, innerWidth))
	if openInGhostty {
		nameLine += " " + currentStyles.panelSelected.Render("●")
	}
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		nameLine,
		currentStyles.path.Render(truncate(project.Path, innerWidth)),
		"",
		sourceLine,
	)

	style := fitOuterWidth(normalStyle, outerWidth).Height(cardHeight)
	if selected {
		style = fitOuterWidth(activeStyle, outerWidth).Height(cardHeight)
	}

	return style.Render(body)
}

func overlayTop(base, overlay string) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")
	if len(baseLines) == 0 {
		return overlay
	}

	out := make([]string, 0, max(len(baseLines), len(overlayLines)))
	for i := 0; i < len(baseLines); i++ {
		if i < len(overlayLines) {
			out = append(out, overlayLines[i])
			continue
		}
		out = append(out, baseLines[i])
	}

	for i := len(baseLines); i < len(overlayLines); i++ {
		out = append(out, overlayLines[i])
	}

	return strings.Join(out, "\n")
}

func clampLines(value string, maxLines int) string {
	if maxLines <= 0 || value == "" {
		return ""
	}
	lines := strings.Split(value, "\n")
	if len(lines) <= maxLines {
		return value
	}
	return strings.Join(lines[:maxLines], "\n")
}

func isLeakedTerminalSequence(value string) bool {
	if value == "" {
		return false
	}
	if hasMouseProtocolFragment(value) {
		return true
	}
	if !strings.HasSuffix(value, "u") {
		return false
	}
	body := strings.TrimSuffix(value, "u")
	parts := strings.Split(body, ";")
	if len(parts) < 2 || len(parts) > 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

func hasMouseProtocolFragment(value string) bool {
	if value == "" {
		return false
	}
	for _, marker := range []string{"[<", "\x1b[<"} {
		idx := strings.Index(value, marker)
		if idx < 0 {
			continue
		}
		fragment := value[idx+len(marker):]
		if fragment == "" {
			return true
		}
		valid := true
		for _, r := range fragment {
			if (r >= '0' && r <= '9') || r == ';' || r == 'M' || r == 'm' {
				continue
			}
			valid = false
			break
		}
		if valid {
			return true
		}
	}
	return false
}

func stripLeakedTerminalFragments(value string) string {
	if value == "" {
		return value
	}
	for _, marker := range []string{"\x1b[<", "[<"} {
		for {
			idx := strings.Index(value, marker)
			if idx < 0 {
				break
			}
			end := idx + len(marker)
			for end < len(value) {
				ch := value[end]
				if (ch >= '0' && ch <= '9') || ch == ';' || ch == 'M' || ch == 'm' {
					end++
					continue
				}
				break
			}
			value = value[:idx] + value[end:]
		}
	}
	return value
}

func renderCommandPanel(m model, width, maxLines int) string {
	entries := m.filteredCommands()
	panelWidth := width
	panel, panelContentWidth := fitPanel(currentStyles.panel, panelWidth)
	header := []string{}
	if maxLines >= 8 && panelContentWidth >= 72 {
		header = append(header, renderSingleLine("  ", "Commands", panelContentWidth, currentStyles.panelTitle))
		header = append(header, renderSingleLine("  ", "↑↓ select  ·  enter run  ·  esc cancel", panelContentWidth, currentStyles.panelSubtitle))
		header = append(header, "")
	} else if maxLines >= 6 && panelContentWidth >= 28 {
		header = append(header, renderSingleLine("  ", "Commands", panelContentWidth, currentStyles.panelTitle))
		header = append(header, "")
	}

	if len(entries) == 0 {
		return panel.Render(strings.Join(append(header, renderSingleLine("  ", "No matching commands.", panelContentWidth, currentStyles.subtle)), "\n"))
	}

	total := len(entries)
	cursor := m.cmdCursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = total - 1
	}

	bodyBudget := maxLines - panel.GetVerticalFrameSize() - len(header)
	if bodyBudget < 1 {
		bodyBudget = 1
	}
	start := cursor - bodyBudget/2
	if start < 0 {
		start = 0
	}
	end := start + bodyBudget
	if end > total {
		end = total
		start = max(0, end-bodyBudget)
	}

	topHint := start > 0
	bottomHint := end < total
	for topHint || bottomHint {
		hintLines := 0
		if topHint {
			hintLines++
		}
		if bottomHint {
			hintLines++
		}
		if end-start+hintLines <= bodyBudget {
			break
		}
		if bottomHint && end > cursor+1 {
			end--
		} else if topHint && start < cursor {
			start++
		} else {
			break
		}
		topHint = start > 0
		bottomHint = end < total
	}

	rows := make([]string, 0, bodyBudget)
	if topHint {
		rows = append(rows, renderSingleLine("  ", fmt.Sprintf("↑ %d more", start), panelContentWidth, currentStyles.subtle))
	}
	selectedPrefixWidth := lipgloss.Width("> ")
	defaultPrefixWidth := lipgloss.Width("  ")
	showDesc := panelContentWidth >= 64
	for i := start; i < end; i++ {
		e := entries[i]
		prefix := "  "
		width := panelContentWidth + defaultPrefixWidth
		rowText := "/" + e.name
		if showDesc {
			rowText += "  " + e.desc
		}
		if i == cursor {
			prefix = currentStyles.panelSelected.Render("> ")
			width = panelContentWidth + selectedPrefixWidth
			rows = append(rows, renderSingleLine(prefix, rowText, width, currentStyles.listSelected.Copy().Bold(true)))
			continue
		}
		rows = append(rows, renderSingleLine(prefix, rowText, width, currentStyles.command.Copy().Bold(true).Foreground(lipgloss.Color(activeTheme.Palette.Accent))))
	}
	if bottomHint {
		rows = append(rows, renderSingleLine("  ", fmt.Sprintf("↓ %d more", total-end), panelContentWidth, currentStyles.subtle))
	}

	return panel.Render(strings.Join(append(header, rows...), "\n"))
}

func renderSettingsPanel(m model, width, maxLines int) string {
	panelWidth := width
	if width >= 28 {
		panelWidth = width - 4
	}
	panelStyle := currentStyles.panel.BorderForeground(lipgloss.Color(activeTheme.Palette.BorderActive))
	panel, panelContentWidth := fitPanel(panelStyle, panelWidth)
	compact := panelContentWidth < 52

	labelWidth := 12
	if compact {
		labelWidth = max(8, min(12, panelContentWidth/3))
	}
	labelStyle := currentStyles.listNormal.Width(labelWidth)
	selectedLabelStyle := currentStyles.listSelected.Copy().Width(labelWidth)
	valueStyle := currentStyles.listSelectedAlt
	activeValueStyle := currentStyles.listSelected.Copy().Bold(true)
	arrowStyle := currentStyles.subtle
	selectedArrowStyle := currentStyles.panelSelected
	cursorStyle := currentStyles.panelSelected

	renderRow := func(idx int, label string, options []string, current string) string {
		selected := idx == m.settingsCursor
		lbl := labelStyle.Render(label)
		if selected {
			lbl = selectedLabelStyle.Render(label)
		}

		arrow := arrowStyle
		if selected {
			arrow = selectedArrowStyle
		}

		parts := []string{arrow.Render("←  ")}
		for i, opt := range options {
			if i > 0 {
				parts = append(parts, valueStyle.Render("  ·  "))
			}
			if opt == current {
				parts = append(parts, activeValueStyle.Render(opt))
			} else {
				parts = append(parts, valueStyle.Render(opt))
			}
		}
		parts = append(parts, arrow.Render("  →"))

		cursor := "  "
		if selected {
			cursor = cursorStyle.Render("▶ ")
		}

		if compact {
			rowLabel := cursor + lbl
			if selected {
				rowLabel = cursor + currentStyles.listSelected.Copy().Bold(true).Render(label)
			} else {
				rowLabel = cursor + currentStyles.listNormal.Render(label)
			}
			optionsLine := "   " + arrow.Render("←  ") + activeValueStyle.Render(current) + arrow.Render("  →")
			return rowLabel + "\n" + optionsLine
		}

		return cursor + lbl + lipgloss.JoinHorizontal(lipgloss.Left, parts...)
	}

	showTimeVal := "off"
	if m.showLastOpened {
		showTimeVal = "on"
	}
	ghosttyOpen := m.settings.Ghostty.Open
	if ghosttyOpen == "" {
		ghosttyOpen = "off"
	}
	ghosttyLayout := m.settings.Ghostty.Layout
	if ghosttyLayout == "" {
		ghosttyLayout = "off"
	}
	ghosttyIndicator := "off"
	if m.settings.Ghostty.Indicator {
		ghosttyIndicator = "on"
	}
	finderReveal := m.finderRevealMode()
	type settingsBlock struct {
		row  int
		text string
	}

	blocks := []settingsBlock{
		{row: -1, text: "  " + currentStyles.subtle.Render("Core")},
		{row: 0, text: renderRow(0, "Sort by", []string{"name", "recent"}, m.sortMode)},
		{row: 1, text: renderRow(1, "Display", []string{"grid", "list"}, m.displayMode)},
		{row: 2, text: renderRow(2, "Grid width", []string{"compact", "normal", "wide"}, m.gridWidthMode())},
		{row: 3, text: renderRow(3, "Provider counts", []string{"off", "on"}, boolToOnOff(m.showProviderCounts()))},
		{row: 4, text: renderRow(4, "Show time", []string{"off", "on"}, showTimeVal)},
		{row: 5, text: renderRow(5, "Theme", themeFamilyIDs(), m.themeFamily())},
		{row: 6, text: renderRow(6, "Mode", []string{"auto", "dark", "light"}, m.themeMode())},
		{row: 7, text: renderRow(7, "Data", []string{"tableau10"}, m.dataPalette())},
		{row: -1, text: "  " + currentStyles.subtle.Render("Ghostty")},
		{row: 8, text: renderRow(8, "Open", []string{"off", "new-window"}, ghosttyOpen)},
		{row: 9, text: renderRow(9, "Layout", []string{"off", "shell", "dev", "ai"}, ghosttyLayout)},
		{row: 10, text: renderRow(10, "Indicator", []string{"off", "on"}, ghosttyIndicator)},
		{row: -1, text: "  " + currentStyles.subtle.Render("Finder")},
		{row: 11, text: renderRow(11, "Reveal", []string{"reveal", "open"}, finderReveal)},
	}

	header := []string{"  " + currentStyles.panelTitle.Render("Settings")}
	footer := []string{}
	if panelContentWidth >= 56 {
		footer = append(footer, renderSingleLine("  ", "↑↓ navigate  ·  ←→ change  ·  /theme browser  ·  esc/enter close", panelContentWidth, currentStyles.subtle))
	} else if panelContentWidth >= 34 {
		footer = append(footer, renderSingleLine("  ", "↑↓ navigate  ·  ←→ change  ·  esc close", panelContentWidth, currentStyles.subtle))
	}

	available := maxLines - panelStyle.GetVerticalBorderSize() - len(header) - len(footer)
	if available < 1 {
		available = 1
	}

	selectedBlock := 0
	for i, block := range blocks {
		if block.row == m.settingsCursor {
			selectedBlock = i
			break
		}
	}

	blockHeight := func(block settingsBlock) int {
		return len(strings.Split(block.text, "\n"))
	}
	totalHeight := func(start, end int) int {
		total := 0
		for i := start; i < end; i++ {
			total += blockHeight(blocks[i])
		}
		return total
	}

	start, end := 0, len(blocks)
	for totalHeight(start, end) > available && start < end {
		leftDistance := selectedBlock - start
		rightDistance := (end - 1) - selectedBlock
		switch {
		case rightDistance > leftDistance && end-1 > selectedBlock:
			end--
		case start < selectedBlock:
			start++
		case end-1 > selectedBlock:
			end--
		default:
			break
		}
		if totalHeight(start, end) <= available {
			break
		}
		if start == selectedBlock && end == selectedBlock+1 {
			break
		}
	}

	lines := append([]string{}, header...)
	if start > 0 {
		lines = append(lines, "  "+currentStyles.subtle.Render("↑ more"))
	}
	for i := start; i < end; i++ {
		lines = append(lines, strings.Split(blocks[i].text, "\n")...)
	}
	if end < len(blocks) {
		lines = append(lines, "  "+currentStyles.subtle.Render("↓ more"))
	}
	lines = append(lines, footer...)

	return panel.Render(strings.Join(lines, "\n"))
}

func renderLoadingProjectsPanel(m model, width int) string {
	if m.displayMode == "list" {
		return renderLoadingList(m, width)
	}
	return renderLoadingGrid(m, width)
}

func renderLoadingGrid(m model, width int) string {
	width = max(1, width)
	cardStyle := m.cardStyle()
	columns := m.gridColumns()
	rows := m.gridRows()
	cardTotalHeight := cardHeight + cardStyle.GetVerticalFrameSize()
	columnWidths := gridColumnWidths(width, columns)

	spacerLines := make([]string, cardTotalHeight)
	for i := range spacerLines {
		spacerLines[i] = " "
	}
	colSpacer := strings.Join(spacerLines, "\n")

	renderedRows := make([]string, 0, rows)
	for row := 0; row < rows; row++ {
		cells := make([]string, 0, columns*2-1)
		for col := 0; col < columns; col++ {
			if col > 0 {
				cells = append(cells, colSpacer)
			}
			frame := m.loadingProjectsFrame + (row * columns) + col
			cells = append(cells, renderLoadingCard(columnWidths[col], frame, cardStyle))
		}
		renderedRows = append(renderedRows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}
	return strings.Join(renderedRows, "\n")
}

func renderLoadingCard(outerWidth, frame int, style lipgloss.Style) string {
	style = fitOuterWidth(style, outerWidth).Height(cardHeight)
	innerWidth := max(1, outerWidth-style.GetHorizontalFrameSize())

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		renderLoadingBar(innerWidth, frame, 0.58, 8),
		renderLoadingBar(innerWidth, frame+1, 0.82, 14),
		"",
		renderLoadingSourceLine(innerWidth, frame+2),
	)

	return style.Render(body)
}

func renderLoadingList(m model, width int) string {
	width = max(1, width)
	rows := m.listRows()
	lines := make([]string, 0, rows)

	showPath := width >= 52
	showSource := width >= 38
	showTime := m.showLastOpened && width >= 72

	timeWidth := 0
	if showTime {
		timeWidth = 10
	}
	separatorCount := 0
	if showPath {
		separatorCount++
	}
	if showSource {
		separatorCount++
	}
	if showTime {
		separatorCount++
	}

	available := width - 2 - (separatorCount * 2)
	if available < 1 {
		available = 1
	}
	if showTime {
		available -= timeWidth
	}

	sourceWidth := 0
	if showSource {
		sourceWidth = min(18, max(10, available/4))
		available -= sourceWidth
	}

	nameWidth := min(24, max(12, available/3))
	if !showPath {
		nameWidth = max(1, available)
	}
	if nameWidth > available {
		nameWidth = available
	}

	pathWidth := 0
	if showPath {
		pathWidth = available - nameWidth
		if pathWidth < 12 {
			showPath = false
			pathWidth = 0
			nameWidth = max(1, available)
		}
	}

	for i := 0; i < rows; i++ {
		frame := m.loadingProjectsFrame + i
		parts := []string{
			"  " + lipgloss.NewStyle().Width(nameWidth).Render(renderLoadingBar(nameWidth, frame, 0.62, 10)),
		}
		if showPath {
			parts = append(parts, "  ")
			parts = append(parts, lipgloss.NewStyle().Width(pathWidth).Render(renderLoadingBar(pathWidth, frame+1, 0.86, 18)))
		}
		if showSource {
			parts = append(parts, "  ")
			parts = append(parts, lipgloss.NewStyle().Width(sourceWidth).Render(renderLoadingBar(sourceWidth, frame+2, 0.54, 8)))
		}
		if showTime {
			parts = append(parts, "  ")
			parts = append(parts, lipgloss.NewStyle().Width(timeWidth).Render(renderLoadingBar(timeWidth, frame+3, 0.48, 4)))
		}
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, parts...))
	}

	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

func renderLoadingSourceLine(width, frame int) string {
	if width < 8 {
		return renderLoadingBar(width, frame, 0.64, 4)
	}

	leftWidth := max(4, min(width-3, int(float64(width)*0.52)))
	rightWidth := width - leftWidth - 2
	if rightWidth < 3 {
		return renderLoadingBar(width, frame, 0.64, 4)
	}

	left := renderLoadingBar(leftWidth, frame, 0.72, 6)
	right := lipgloss.NewStyle().Align(lipgloss.Right).Width(rightWidth).Render(renderLoadingBar(rightWidth, frame+1, 0.58, 3))
	return left + "  " + right
}

func renderLoadingBar(width, frame int, ratio float64, minWidth int) string {
	width = max(1, width)
	target := int(float64(width) * ratio)
	if target < minWidth {
		target = minWidth
	}
	if target > width {
		target = width
	}

	phase := frame % 3
	switch phase {
	case 1:
		target = max(1, target-2)
	case 2:
		target = min(width, target+1)
	}

	if target <= 2 {
		return currentStyles.subtle.Render(strings.Repeat("·", target))
	}

	if target == 3 {
		return currentStyles.subtle.Render("···")
	}

	coreWidth := target - 2
	highlightWidth := min(3, max(1, coreWidth/4))
	highlightStart := 0
	if coreWidth-highlightWidth > 0 {
		highlightStart = frame % (coreWidth - highlightWidth + 1)
	}

	leftCap := "╶"
	rightCap := "╴"
	left := strings.Repeat("╌", highlightStart)
	mid := strings.Repeat("╍", highlightWidth)
	right := strings.Repeat("╌", max(0, coreWidth-highlightStart-highlightWidth))

	return currentStyles.subtle.Render(leftCap+left) +
		currentStyles.panelSelected.Render(mid) +
		currentStyles.subtle.Render(right+rightCap)
}

func pingPong(step, maxValue int) int {
	if maxValue <= 0 {
		return 0
	}
	period := maxValue * 2
	if period == 0 {
		return 0
	}
	pos := step % period
	if pos > maxValue {
		return period - pos
	}
	return pos
}

func renderThemePickerPanel(m model, width int) string {
	families := uitheme.Families()
	panelWidth := width
	if width >= 48 {
		panelWidth = width - 4
	}
	panelWidth = min(82, panelWidth)
	panelStyle := currentStyles.panel.BorderForeground(lipgloss.Color(activeTheme.Palette.BorderActive))
	panel, panelContentWidth := fitPanel(panelStyle, panelWidth)
	compact := panelContentWidth < 52

	lines := []string{renderSingleLine("  ", "Theme Browser", panelContentWidth, currentStyles.panelTitle)}
	if panelContentWidth >= 60 {
		lines = append(lines, renderSingleLine("  ", "Move with up/down for live preview. Enter applies, esc restores.", panelContentWidth, currentStyles.panelSubtitle))
		lines = append(lines, "")
	} else if panelContentWidth >= 36 {
		lines = append(lines, renderSingleLine("  ", "↑↓ preview  ·  enter apply  ·  esc cancel", panelContentWidth, currentStyles.panelSubtitle))
		lines = append(lines, "")
	}

	for i, family := range families {
		prefix := "  "
		content := family.Name
		if panelContentWidth >= 44 {
			content += "  " + family.Description
		}
		if i == m.themePickerCursor {
			prefix = currentStyles.panelSelected.Render("> ")
			lines = append(lines, renderSingleLine(prefix, content, panelContentWidth+lipgloss.Width(prefix), currentStyles.panelSelected))
			continue
		}
		if compact {
			content = family.Name
		}
		lines = append(lines, renderSingleLine(prefix, content, panelContentWidth+lipgloss.Width(prefix), currentStyles.listSelected.Copy().Bold(true)))
	}

	lines = append(lines, "")
	lines = append(lines, renderSingleLine("  ", "Mode", panelContentWidth, currentStyles.listNormal))
	lines = append(lines, renderSingleLine("  ", m.themeMode(), panelContentWidth, currentStyles.listSelectedAlt))
	lines = append(lines, "")
	lines = append(lines, renderSingleLine("  ", "Data Palette", panelContentWidth, currentStyles.listNormal))
	lines = append(lines, renderSingleLine("  ", m.dataPalette(), panelContentWidth, currentStyles.listSelectedAlt))
	lines = append(lines, "")
	lines = append(lines, renderThemePreviewStrip(m.resolvedTheme.Palette, panelContentWidth))

	return panel.Render(strings.Join(lines, "\n"))
}

func renderOnboardingScreen(m model) string {
	contentWidth := m.contentWidth()
	panelStyle := currentStyles.panel.BorderForeground(lipgloss.Color(activeTheme.Palette.BorderActive))
	panel, panelContentWidth := fitPanel(panelStyle, contentWidth)
	availableHeight := m.height - (m.pagePaddingY() * 2)
	if availableHeight < 10 {
		availableHeight = 10
	}

	sourceOffset := 1
	var lines []string
	if m.onboardingIsProviders {
		sourceOffset = 0
		lines = []string{
			renderSingleLine("  ", "Providers", panelContentWidth, currentStyles.panelTitle),
			renderSingleLine("  ", "Enable or disable project sources", panelContentWidth, currentStyles.panelSubtitle),
			"",
		}
	} else {
		lines = []string{
			renderSingleLine("  ", "Welcome to RepoDock", panelContentWidth, currentStyles.panelTitle),
			renderSingleLine("  ", "First-run setup", panelContentWidth, currentStyles.panelSubtitle),
			"",
			renderSingleLine("  ", "Choose your default layout and which project sources to import.", panelContentWidth, currentStyles.listSelectedAlt),
			"",
		}

		displayLabel := "Default display"
		displayValue := m.onboardingDisplay
		if displayValue == "" {
			displayValue = "grid"
		}
		displayPrefix := "  "
		displayStyle := currentStyles.listNormal
		if m.onboardingCursor == 0 {
			displayPrefix = currentStyles.panelSelected.Render("▶ ")
			displayStyle = currentStyles.listSelected.Copy().Bold(true)
		}
		lines = append(lines, renderSingleLine(displayPrefix, displayLabel+"  ←  "+displayValue+"  →", panelContentWidth+lipgloss.Width(displayPrefix), displayStyle))
		lines = append(lines, "")
	}

	lines = append(lines, renderSingleLine("  ", "Import sources", panelContentWidth, currentStyles.panelTitle))

	for i, src := range m.onboardingSources {
		prefix := "  "
		style := currentStyles.listNormal
		if m.onboardingCursor == i+sourceOffset {
			prefix = currentStyles.panelSelected.Render("▶ ")
			style = currentStyles.listSelected.Copy().Bold(true)
		}
		state := "off"
		if src.Enabled {
			state = "on"
		}
		status := string(src.Status)
		if src.Available {
			status = "available"
		}
		line := fmt.Sprintf("%s  %s  (%s)", src.Name, state, status)
		lines = append(lines, renderSingleLine(prefix, line, panelContentWidth+lipgloss.Width(prefix), style))
	}

	closeLabel := "Finish setup"
	if m.onboardingIsProviders {
		closeLabel = "Save and close"
	}
	lines = append(lines, "")
	finishPrefix := "  "
	finishStyle := currentStyles.listNormal
	if m.onboardingCursor == m.onboardingRowCount()-1 {
		finishPrefix = currentStyles.panelSelected.Render("▶ ")
		finishStyle = currentStyles.listSelected.Copy().Bold(true)
	}
	lines = append(lines, renderSingleLine(finishPrefix, closeLabel, panelContentWidth+lipgloss.Width(finishPrefix), finishStyle))
	lines = append(lines, "")
	lines = append(lines, renderSingleLine("  ", "↑↓ select  ·  ←→ change  ·  enter apply/finish  ·  esc skip", panelContentWidth, currentStyles.subtle))

	contentHeight := availableHeight - panelStyle.GetVerticalBorderSize()
	if contentHeight < 1 {
		contentHeight = 1
	}
	if len(lines) < contentHeight {
		for len(lines) < contentHeight {
			lines = append(lines, "")
		}
	} else if len(lines) > contentHeight {
		lines = append(lines[:max(0, contentHeight-1)], renderSingleLine("  ", "...", panelContentWidth, currentStyles.subtle))
	}

	page := currentStyles.page.Copy().Padding(m.pagePaddingY(), m.pagePaddingX())
	return page.Render(panel.Height(contentHeight).Render(strings.Join(lines, "\n")))
}

func renderActionPanel(m model, width int) string {
	if m.themePickerOpen || m.newProjectOpen || m.settingsOpen || m.hiddenOpen || m.layoutOpen || !m.actionOpen || m.isCommandMode() {
		return ""
	}

	actions := m.projectActions()
	if len(actions) == 0 {
		return ""
	}

	project, _ := m.selectedProject()
	panelWidth := width
	if width >= 40 {
		panelWidth = width - 4
	}
	panelWidth = min(64, panelWidth)
	panel, panelContentWidth := fitPanel(currentStyles.panel, panelWidth)

	lines := []string{
		currentStyles.panelTitle.Render("Project Actions"),
		currentStyles.panelSubtitle.Render(project.Name),
		"",
	}

	for i, action := range actions {
		selected := i == m.actionCursor
		prefixText := "  "
		prefix := prefixText
		labelStyle := currentStyles.listSelected.Copy().Bold(true)
		descStyle := currentStyles.listSelectedAlt
		if selected {
			prefixText = "> "
			prefix = currentStyles.panelSelected.Render(prefixText)
			labelStyle = currentStyles.panelSelected.Copy().Bold(true)
			descStyle = currentStyles.listSelectedAlt
		}

		available := panelContentWidth - lipgloss.Width(prefixText)
		if available < 1 {
			available = 1
		}
		line := prefix + labelStyle.Render(truncate(action.label, available))
		if available >= 26 && strings.TrimSpace(action.desc) != "" {
			descAvail := available - lipgloss.Width(action.label) - 2
			if descAvail >= 8 {
				line += "  " + descStyle.Render(truncate(action.desc, descAvail))
			}
		}
		lines = append(lines, line)
	}

	for i := range lines {
		if i < 2 {
			lines[i] = renderSingleLine("", lines[i], panelContentWidth, lipgloss.NewStyle())
		}
	}
	return panel.Render(strings.Join(lines, "\n"))
}

func renderHiddenPanel(m model, width int, maxLines int) string {
	panelWidth := width
	if width >= 44 {
		panelWidth = width - 4
	}
	panelWidth = min(72, panelWidth)
	panelStyle := currentStyles.panel.BorderForeground(lipgloss.Color(activeTheme.Palette.BorderActive))
	panel, panelContentWidth := fitPanel(panelStyle, panelWidth)

	paths := m.hiddenProjectPaths()
	lines := []string{
		renderSingleLine("  ", "Hidden Projects", panelContentWidth, currentStyles.panelTitle),
		renderSingleLine("  ", fmt.Sprintf("%d hidden", len(paths)), panelContentWidth, currentStyles.panelSubtitle),
		"",
	}

	if len(paths) == 0 {
		lines = append(lines, renderSingleLine("  ", "No hidden projects.", panelContentWidth, currentStyles.listNormal))
		lines = append(lines, "")
		lines = append(lines, renderSingleLine("  ", "Esc closes this panel.", panelContentWidth, currentStyles.panelSubtitle))
		return clampLines(panel.Render(strings.Join(lines, "\n")), maxLines)
	}

	availableRows := maxLines - len(lines) - 2
	if availableRows < 1 {
		availableRows = 1
	}
	start := 0
	if m.hiddenCursor >= availableRows {
		start = m.hiddenCursor - availableRows + 1
	}
	end := min(len(paths), start+availableRows)

	for i := start; i < end; i++ {
		path := paths[i]
		name := filepath.Base(path)
		if name == "." || name == string(filepath.Separator) || name == "" {
			name = path
		}
		rowText := name + "  " + path
		prefix := "  "
		style := currentStyles.listNormal
		if i == m.hiddenCursor {
			prefix = currentStyles.panelSelected.Render("> ")
			style = currentStyles.listSelected.Copy().Bold(true)
		}
		lines = append(lines, renderSingleLine(prefix, rowText, panelContentWidth+lipgloss.Width(prefix), style))
	}

	if end < len(paths) {
		lines = append(lines, renderSingleLine("  ", fmt.Sprintf("… %d more", len(paths)-end), panelContentWidth, currentStyles.panelSubtitle))
	} else {
		lines = append(lines, "")
	}
	lines = append(lines, renderSingleLine("  ", "Enter restores selected project.", panelContentWidth, currentStyles.panelSubtitle))

	return clampLines(panel.Render(strings.Join(lines, "\n")), maxLines)
}

func renderNewProjectPanel(m model, width, maxLines int) string {
	panelWidth := width
	if width >= 28 {
		panelWidth = width - 4
	}
	panelStyle := currentStyles.panel.BorderForeground(lipgloss.Color(activeTheme.Palette.BorderActive))
	panel, panelContentWidth := fitPanel(panelStyle, panelWidth)

	lines := []string{
		renderSingleLine("  ", "Import Project", panelContentWidth, currentStyles.panelTitle),
		renderSingleLine("  ", "Drag a folder into the path field, or paste an absolute path.", panelContentWidth, currentStyles.panelSubtitle),
		"",
	}

	pathLabel := currentStyles.listNormal.Render("Path")
	nameLabel := currentStyles.listNormal.Render("Name")
	if m.newProjectCursor == 0 {
		pathLabel = currentStyles.listSelected.Copy().Bold(true).Render("Path")
	}
	if m.newProjectCursor == 1 {
		nameLabel = currentStyles.listSelected.Copy().Bold(true).Render("Name")
	}

	lines = append(lines, "  "+pathLabel)
	lines = append(lines, "  "+m.newProjectPathInput.View())
	lines = append(lines, "")
	lines = append(lines, "  "+nameLabel)
	lines = append(lines, "  "+m.newProjectNameInput.View())
	lines = append(lines, "")
	lines = append(lines, renderSingleLine("  ", "Press Enter to import. The name is optional and only changes the RepoDock label.", panelContentWidth, currentStyles.subtle))

	return clampLines(panel.Render(strings.Join(lines, "\n")), maxLines)
}

func (m *model) openOnboarding() {
	m.onboardingOpen = true
	m.onboardingIsProviders = false
	m.actionOpen = false
	m.viewFocus = false
	m.onboardingCursor = 0
	m.syncOnboardingState()
	m.status = "Onboarding open."
}

func (m *model) openProviders() {
	m.onboardingOpen = true
	m.onboardingIsProviders = true
	m.actionOpen = false
	m.viewFocus = false
	m.onboardingCursor = 0
	m.syncOnboardingState()
	m.status = "Providers open."
}

func (m *model) openNewProjectImport() {
	m.newProjectOpen = true
	m.newProjectCursor = 0
	m.input.SetValue("")
	m.cmdCursor = 0
	m.newProjectPathInput.SetValue("")
	m.newProjectNameInput.SetValue("")
	m.syncNewProjectFocus()
	m.status = "Import a manual project."
}

func (m *model) closeNewProjectImport() {
	m.newProjectOpen = false
	m.newProjectCursor = 0
	m.newProjectPathInput.Blur()
	m.newProjectNameInput.Blur()
	m.newProjectPathInput.SetValue("")
	m.newProjectNameInput.SetValue("")
	m.status = "Import project closed."
}

func (m *model) syncNewProjectFocus() {
	if m.newProjectCursor <= 0 {
		m.newProjectCursor = 0
		m.newProjectNameInput.Blur()
		m.newProjectPathInput.Focus()
		return
	}
	m.newProjectCursor = 1
	m.newProjectPathInput.Blur()
	m.newProjectNameInput.Focus()
}

func normalizeImportedPath(raw string) string {
	path := strings.TrimSpace(raw)
	path = strings.Trim(path, "\"'")
	replacer := strings.NewReplacer(
		`\\ `, `\ `,
		`\ `, ` `,
		`\"`, `"`,
		`\'`, `'`,
	)
	path = replacer.Replace(path)
	return filepath.Clean(path)
}

func (m *model) finishNewProjectImport() tea.Cmd {
	path := normalizeImportedPath(m.newProjectPathInput.Value())
	name := strings.TrimSpace(m.newProjectNameInput.Value())
	cmd := m.addManualProjectWithName(path, name)
	if cmd == nil {
		return nil
	}
	m.newProjectOpen = false
	m.newProjectCursor = 0
	m.newProjectPathInput.Blur()
	m.newProjectNameInput.Blur()
	m.newProjectPathInput.SetValue("")
	m.newProjectNameInput.SetValue("")
	return cmd
}

func (m *model) closeOnboarding() tea.Cmd {
	m.onboardingOpen = false
	if m.onboardingIsProviders {
		m.syncCursor()
		m.status = "Providers saved."
		return saveProviderConfigCmd(m.providerConfigStore, m.buildProviderConfig(), true)
	}
	m.settings.Onboarding.Seen = true
	m.displayMode = m.onboardingDisplay
	m.syncCursor()
	m.status = "Onboarding closed."
	return tea.Batch(
		saveSettingsCmd(m.settingsStore, m.settings, "Onboarding closed."),
		saveProviderConfigCmd(m.providerConfigStore, m.buildProviderConfig(), true),
		saveStateCmd(m.stateStore, m.buildAppState(), ""),
	)
}

func (m model) onboardingRowCount() int {
	if m.onboardingIsProviders {
		return len(m.onboardingSources) + 1 // sources + close button
	}
	return 1 + len(m.onboardingSources) + 1 // display mode + sources + finish button
}

func (m *model) syncOnboardingState() {
	if m.displayMode == "list" {
		m.onboardingDisplay = "list"
	} else {
		m.onboardingDisplay = "grid"
	}

	preserved := make(map[string]onboardingSource, len(m.onboardingSources))
	for _, src := range m.onboardingSources {
		preserved[src.ID] = src
	}

	config, _ := m.providerConfigStore.Load()
	enabledByID := make(map[string]bool, len(config.Providers))
	for _, entry := range config.Providers {
		id := strings.TrimSpace(strings.ToLower(entry.ID))
		if id == "" || entry.Enabled == nil {
			continue
		}
		enabledByID[id] = *entry.Enabled
	}

	next := make([]onboardingSource, 0, max(len(m.providers), 2))
	if len(m.providers) > 0 {
		for _, detection := range m.providers {
			id := strings.TrimSpace(strings.ToLower(detection.ID))
			if id == "" {
				continue
			}
			src := onboardingSource{
				ID:        id,
				Name:      detection.Name,
				Enabled:   detection.Enabled,
				Available: detection.Available,
				Status:    detection.Status,
			}
			if src.Name == "" {
				src.Name = id
			}
			if enabled, ok := enabledByID[id]; ok {
				src.Enabled = enabled
			}
			if prev, ok := preserved[id]; ok {
				src.Enabled = prev.Enabled
			}
			next = append(next, src)
		}
	}

	if len(next) == 0 {
		for _, id := range []string{"codex", "claude"} {
			src := onboardingSource{
				ID:        id,
				Name:      id,
				Enabled:   true,
				Available: false,
				Status:    sources.StatusMissing,
			}
			if enabled, ok := enabledByID[id]; ok {
				src.Enabled = enabled
			}
			if prev, ok := preserved[id]; ok {
				src.Enabled = prev.Enabled
				src.Available = prev.Available
				if prev.Status != "" {
					src.Status = prev.Status
				}
			}
			next = append(next, src)
		}
	}

	m.onboardingSources = next
	if m.onboardingCursor >= m.onboardingRowCount() {
		m.onboardingCursor = m.onboardingRowCount() - 1
	}
	if m.onboardingCursor < 0 {
		m.onboardingCursor = 0
	}
}

func (m *model) moveOnboarding(delta int) {
	rows := m.onboardingRowCount()
	if rows <= 0 {
		m.onboardingCursor = 0
		return
	}
	m.onboardingCursor += delta
	if m.onboardingCursor < 0 {
		m.onboardingCursor = rows - 1
	}
	if m.onboardingCursor >= rows {
		m.onboardingCursor = 0
	}
}

func (m *model) adjustOnboarding(delta int) {
	// In providers mode: cursor 0..N-1 = sources, cursor N = close button
	// In onboarding mode: cursor 0 = display, cursor 1..N = sources, cursor N+1 = finish
	sourceOffset := 1
	if m.onboardingIsProviders {
		sourceOffset = 0
	}

	sourceIndex := m.onboardingCursor - sourceOffset
	isDisplayRow := !m.onboardingIsProviders && m.onboardingCursor == 0
	isSourceRow := sourceIndex >= 0 && sourceIndex < len(m.onboardingSources)

	switch {
	case isDisplayRow:
		if delta == 0 {
			if m.onboardingDisplay == "list" {
				m.onboardingDisplay = "grid"
			} else {
				m.onboardingDisplay = "list"
			}
		} else if delta < 0 {
			m.onboardingDisplay = "grid"
		} else {
			m.onboardingDisplay = "list"
		}
		m.status = fmt.Sprintf("Default display: %s.", m.onboardingDisplay)
	case isSourceRow:
		enabled := delta > 0
		if delta == 0 {
			enabled = !m.onboardingSources[sourceIndex].Enabled
		}
		m.onboardingSources[sourceIndex].Enabled = enabled
		if enabled {
			m.status = fmt.Sprintf("Import from %s enabled.", m.onboardingSources[sourceIndex].Name)
		} else {
			m.status = fmt.Sprintf("Import from %s disabled.", m.onboardingSources[sourceIndex].Name)
		}
	}
}

func (m model) buildProviderConfig() store.ProviderConfig {
	cfg, _ := m.providerConfigStore.Load()
	entries := append([]store.ProviderEntry(nil), cfg.Providers...)
	indexByID := make(map[string]int, len(entries))
	for i, entry := range entries {
		id := strings.TrimSpace(strings.ToLower(entry.ID))
		if id == "" {
			continue
		}
		indexByID[id] = i
	}

	for _, src := range m.onboardingSources {
		id := strings.TrimSpace(strings.ToLower(src.ID))
		if id == "" {
			continue
		}
		enabled := src.Enabled
		if idx, ok := indexByID[id]; ok {
			entries[idx].Enabled = &enabled
			if strings.TrimSpace(entries[idx].Kind) == "" {
				entries[idx].Kind = id
			}
			if strings.TrimSpace(entries[idx].Name) == "" {
				entries[idx].Name = src.Name
			}
			continue
		}
		entries = append(entries, store.ProviderEntry{
			ID:      id,
			Kind:    id,
			Name:    src.Name,
			Enabled: &enabled,
		})
	}

	return store.ProviderConfig{Providers: entries}
}

func saveProviderConfigCmd(configStore store.ProviderConfigStore, cfg store.ProviderConfig, reload bool) tea.Cmd {
	return func() tea.Msg {
		if err := configStore.Save(cfg); err != nil {
			return providerConfigSaveFailedMsg{err: err}
		}
		return providerConfigSavedMsg{reload: reload}
	}
}

func fitOuterWidth(style lipgloss.Style, outerWidth int) lipgloss.Style {
	innerWidth := outerWidth - style.GetHorizontalBorderSize()
	if innerWidth < 0 {
		innerWidth = 0
	}
	return style.Width(innerWidth)
}

func contentWidthForStyle(style lipgloss.Style, outerWidth int) int {
	width := outerWidth - style.GetHorizontalBorderSize() - style.GetHorizontalPadding()
	if width < 1 {
		return 1
	}
	return width
}

func fitPanel(style lipgloss.Style, outerWidth int) (lipgloss.Style, int) {
	return fitOuterWidth(style, outerWidth), contentWidthForStyle(style, outerWidth)
}

func renderSingleLine(prefix, text string, width int, style lipgloss.Style) string {
	available := width - lipgloss.Width(prefix)
	if available < 1 {
		available = 1
	}
	return prefix + style.Render(truncate(text, available))
}

func gridColumnWidths(totalWidth, columns int) []int {
	if columns <= 0 {
		return nil
	}
	if columns == 1 {
		return []int{max(1, totalWidth)}
	}

	usable := totalWidth - (columns - 1)
	if usable < columns {
		usable = columns
	}
	base := usable / columns
	extra := usable % columns
	widths := make([]int, columns)
	for i := 0; i < columns; i++ {
		widths[i] = base
		if i < extra {
			widths[i]++
		}
	}
	return widths
}

func sourceNames(sources []domain.Source) []string {
	names := make([]string, 0, len(sources))
	for _, source := range sources {
		names = append(names, string(source))
	}
	return names
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

func truncate(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= limit {
		return value
	}

	ellipsis := "…"
	ellipsisWidth := lipgloss.Width(ellipsis)
	if limit <= ellipsisWidth {
		return ellipsis
	}

	var b strings.Builder
	width := 0
	for _, r := range value {
		rw := lipgloss.Width(string(r))
		if width+rw+ellipsisWidth > limit {
			break
		}
		b.WriteRune(r)
		width += rw
	}

	if b.Len() == 0 {
		return ellipsis
	}

	return b.String() + ellipsis
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func loadProjectsCmd(stateStore store.AppStateStore) tea.Cmd {
	return func() tea.Msg {
		state, err := stateStore.Load()
		if err != nil {
			return projectsLoadFailedMsg{err: err}
		}

		pinnedPaths := make(map[string]struct{}, len(state.PinnedPaths))
		for _, path := range state.PinnedPaths {
			pinnedPaths[filepath.Clean(path)] = struct{}{}
		}
		hiddenPaths := make(map[string]struct{}, len(state.HiddenPaths))
		for _, path := range state.HiddenPaths {
			hiddenPaths[filepath.Clean(path)] = struct{}{}
		}
		manualProjects := manualProjectsMap(state.ManualProjects)

		lastOpened := activity.LastOpened()

		detections, err := sources.DefaultManager().Detect()
		if err != nil {
			return projectsLoadFailedMsg{err: err}
		}
		providers := sources.AvailableProviders(detections)
		if len(providers) == 0 {
			projects := mergeManualProjects(nil, state.ManualProjects)
			detections = appendManualDetection(detections, state.ManualProjects)
			projects = filterHiddenProjects(projects, hiddenPaths)
			return projectsLoadedMsg{
				projects:       projects,
				manualProjects: manualProjects,
				pinnedPaths:    pinnedPaths,
				hiddenPaths:    hiddenPaths,
				providers:      detections,
				sortMode:       state.SortMode,
				displayMode:    state.DisplayMode,
				lastOpened:     lastOpened,
				showLastOpened: state.ShowLastOpened,
			}
		}

		projects, providerErrs := sources.LoadAll(context.Background(), providers...)
		projects = mergeManualProjects(projects, state.ManualProjects)
		detections = appendManualDetection(detections, state.ManualProjects)
		projects = filterHiddenProjects(projects, hiddenPaths)
		return projectsLoadedMsg{
			projects:       projects,
			manualProjects: manualProjects,
			pinnedPaths:    pinnedPaths,
			hiddenPaths:    hiddenPaths,
			providers:      detections,
			providerErrs:   providerErrs,
			sortMode:       state.SortMode,
			displayMode:    state.DisplayMode,
			lastOpened:     lastOpened,
			showLastOpened: state.ShowLastOpened,
		}
	}
}

func manualProjectsMap(projects []store.ManualProject) map[string]store.ManualProject {
	out := make(map[string]store.ManualProject, len(projects))
	for _, project := range projects {
		cleanPath := filepath.Clean(strings.TrimSpace(project.Path))
		if cleanPath == "" || cleanPath == "." {
			continue
		}
		out[cleanPath] = store.ManualProject{Path: cleanPath, Name: strings.TrimSpace(project.Name)}
	}
	return out
}

func mergeManualProjects(projects []domain.Project, manual []store.ManualProject) []domain.Project {
	if len(manual) == 0 {
		return projects
	}

	merged := append([]domain.Project(nil), projects...)
	indexByPath := make(map[string]int, len(merged))
	for i, project := range merged {
		indexByPath[filepath.Clean(project.Path)] = i
	}

	for _, entry := range manual {
		path := filepath.Clean(strings.TrimSpace(entry.Path))
		if path == "" || path == "." {
			continue
		}
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			name = filepath.Base(path)
			if name == "." || name == string(filepath.Separator) || name == "" {
				name = path
			}
		}

		if idx, ok := indexByPath[path]; ok {
			merged[idx].Sources = appendSourceIfMissing(merged[idx].Sources, domain.SourceManual)
			if strings.TrimSpace(entry.Name) != "" {
				merged[idx].Name = strings.TrimSpace(entry.Name)
			}
			continue
		}

		indexByPath[path] = len(merged)
		merged = append(merged, domain.Project{
			Name:    name,
			Path:    path,
			Sources: []domain.Source{domain.SourceManual},
		})
	}

	return merged
}

func appendSourceIfMissing(current []domain.Source, src domain.Source) []domain.Source {
	for _, existing := range current {
		if existing == src {
			return current
		}
	}
	return append(current, src)
}

func appendManualDetection(detections []sources.Detection, manual []store.ManualProject) []sources.Detection {
	if len(manual) == 0 {
		return detections
	}
	for _, detection := range detections {
		if detection.Name == "manual" {
			return detections
		}
	}
	return append(detections, sources.Detection{
		ID:        "manual",
		Kind:      "manual",
		Name:      "manual",
		Enabled:   true,
		Available: true,
		Status:    sources.StatusAvailable,
	})
}

func (m model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Allow motion events only for hover detection on the view bar
	if msg.Action == tea.MouseActionMotion {
		newHover := ""
		if msg.Y >= m.viewBarStartY() && msg.Y < m.contentStartY() {
			if mode, ok := m.viewModeAt(msg.X, msg.Y); ok {
				newHover = mode
			}
		}
		if newHover != m.hoverViewMode {
			m.hoverViewMode = newHover
			return m, nil
		}
		return m, nil
	}
	// Ignore release; only act on press
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}

	if m.themePickerOpen {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.moveThemePicker(-1)
		case tea.MouseButtonWheelDown:
			m.moveThemePicker(1)
		}
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if !m.actionOpen && !m.isCommandMode() {
			m.scrollViewport(-1)
		}
		return m, nil
	case tea.MouseButtonWheelDown:
		if !m.actionOpen && !m.isCommandMode() {
			m.scrollViewport(1)
		}
		return m, nil
	case tea.MouseButtonLeft:
		if m.isCommandMode() {
			return m, nil
		}

		mode, ok := m.viewModeAt(msg.X, msg.Y)
		if ok {
			m.setViewMode(mode)
			m.hoverViewMode = ""
			return m, nil
		}

		index, ok := m.projectIndexAt(msg.X, msg.Y)
		if !ok {
			return m, nil
		}

		m.cursor = index
		m.syncCursor()
		project, _ := m.selectedProject()
		m.status = fmt.Sprintf("Selected %s", project.Name)
		return m, nil
	default:
		return m, nil
	}
}

func openShellCmd(project domain.Project) tea.Cmd {
	return func() tea.Msg {
		shell := resolveShellPath(os.Getenv("SHELL"))

		cmd := exec.Command(shell)
		cmd.Dir = project.Path
		return tea.ExecProcess(cmd, func(err error) tea.Msg {
			return shellExitedMsg{project: project.Name, err: err}
		})()
	}
}

func projectLaunchTickCmd() tea.Cmd {
	return tea.Tick(90*time.Millisecond, func(time.Time) tea.Msg {
		return projectLaunchTickMsg{}
	})
}

func resolveShellPath(raw string) string {
	shell := strings.TrimSpace(raw)
	if shell == "" || !filepath.IsAbs(shell) {
		return "/bin/zsh"
	}

	info, err := os.Stat(shell)
	if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return "/bin/zsh"
	}

	return shell
}

func queryGhosttyOpenCmd() tea.Cmd {
	return func() tea.Msg {
		return ghosttyOpenChangedMsg{paths: ghostty.OpenProjects()}
	}
}

func openProjectCmd(project domain.Project, settings store.Settings) tea.Cmd {
	open := settings.Ghostty.Open
	if open != "new-window" || !ghostty.IsRunningInGhostty() {
		// fallback: open shell in current terminal
		return openShellCmd(project)
	}
	return func() tea.Msg {
		// Prefer saved layout over preset
		ls := store.NewLayoutStore(project.Path)
		if saved, err := ls.Load(); err == nil && len(saved.Panes) > 0 {
			if err := ghostty.OpenFromLayout(project.Path, saved.Panes); err != nil {
				return shellExitedMsg{project: project.Name, err: err}
			}
			return nil
		}
		// Fall back to preset layout from settings
		preset := settings.Ghostty.Layout
		var err error
		if preset != "" && preset != "shell" && preset != "off" {
			err = ghostty.OpenLayout(project.Path, project.Name, preset)
		} else {
			err = ghostty.OpenWindow(project.Path, project.Name)
		}
		if err != nil {
			return shellExitedMsg{project: project.Name, err: err}
		}
		return nil
	}
}

func copyPathCmd(path string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(path)
		err := cmd.Run()
		return copyPathFinishedMsg{path: path, err: err}
	}
}

func revealInFinderCmd(path, mode string) tea.Cmd {
	return func() tea.Msg {
		if runtime.GOOS != "darwin" {
			return revealInFinderFinishedMsg{path: path, mode: mode, err: fmt.Errorf("finder is only available on macOS")}
		}
		mode = strings.TrimSpace(strings.ToLower(mode))
		var cmd *exec.Cmd
		if mode == "open" {
			cmd = exec.Command("open", path)
		} else {
			mode = "reveal"
			cmd = exec.Command("open", "-R", path)
		}
		err := cmd.Run()
		return revealInFinderFinishedMsg{path: path, mode: mode, err: err}
	}
}

func saveStateCmd(stateStore store.AppStateStore, state store.AppState, action string) tea.Cmd {
	return func() tea.Msg {
		if err := stateStore.Save(state); err != nil {
			return stateSaveFailedMsg{err: err}
		}
		return stateSavedMsg{action: action}
	}
}

func saveSettingsCmd(settingsStore store.SettingsStore, settings store.Settings, action string) tea.Cmd {
	return func() tea.Msg {
		if err := settingsStore.Save(settings); err != nil {
			return settingsSaveFailedMsg{err: err}
		}
		return settingsSavedMsg{action: action, settings: settings}
	}
}

func (m model) buildAppState() store.AppState {
	paths := make([]string, 0, len(m.pinnedPaths))
	for path := range m.pinnedPaths {
		paths = append(paths, path)
	}
	hidden := make([]string, 0, len(m.hiddenPaths))
	for path := range m.hiddenPaths {
		hidden = append(hidden, path)
	}
	manual := make([]store.ManualProject, 0, len(m.manualProjects))
	for _, project := range m.manualProjects {
		manual = append(manual, project)
	}
	lo := m.lastOpened
	if lo == nil {
		lo = make(map[string]time.Time)
	}
	return store.AppState{
		PinnedPaths:    paths,
		HiddenPaths:    hidden,
		ManualProjects: manual,
		SortMode:       m.sortMode,
		DisplayMode:    m.displayMode,
		LastOpened:     lo,
		ShowLastOpened: m.showLastOpened,
	}
}

func sortAndOrderProjects(projects []domain.Project, pinnedPaths map[string]struct{}, sortMode string, lastOpened map[string]time.Time) []domain.Project {
	if len(projects) == 0 {
		return nil
	}

	pinned := make([]domain.Project, 0, len(projects))
	rest := make([]domain.Project, 0, len(projects))
	for _, project := range projects {
		if _, ok := pinnedPaths[filepath.Clean(project.Path)]; ok {
			pinned = append(pinned, project)
		} else {
			rest = append(rest, project)
		}
	}

	cmp := func(a, b domain.Project) bool {
		if sortMode == "recent" && lastOpened != nil {
			ta := lastOpened[filepath.Clean(a.Path)]
			tb := lastOpened[filepath.Clean(b.Path)]
			aOpened := !ta.IsZero()
			bOpened := !tb.IsZero()
			if aOpened || bOpened {
				if aOpened && !bOpened {
					return true
				}
				if !aOpened && bOpened {
					return false
				}
				if !ta.Equal(tb) {
					return ta.After(tb)
				}
			}
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	}

	sort.SliceStable(pinned, func(i, j int) bool { return cmp(pinned[i], pinned[j]) })
	sort.SliceStable(rest, func(i, j int) bool { return cmp(rest[i], rest[j]) })

	return append(pinned, rest...)
}

func filterHiddenProjects(projects []domain.Project, hiddenPaths map[string]struct{}) []domain.Project {
	if len(projects) == 0 || len(hiddenPaths) == 0 {
		return projects
	}

	filtered := make([]domain.Project, 0, len(projects))
	for _, project := range projects {
		if _, hidden := hiddenPaths[filepath.Clean(project.Path)]; hidden {
			continue
		}
		filtered = append(filtered, project)
	}
	return filtered
}

func (m model) isPinned(path string) bool {
	_, ok := m.pinnedPaths[filepath.Clean(path)]
	return ok
}

func (m *model) selectedProject() (domain.Project, bool) {
	projects := m.filteredProjects()
	if len(projects) == 0 {
		return domain.Project{}, false
	}
	if m.cursor < 0 || m.cursor >= len(projects) {
		return domain.Project{}, false
	}
	return projects[m.cursor], true
}

func (m *model) refreshOrderedProjects(selectedPath string) {
	m.projects = sortAndOrderProjects(m.projectsBase, m.pinnedPaths, m.sortMode, m.lastOpened)
	if selectedPath == "" {
		m.syncCursor()
		return
	}

	filtered := m.filteredProjects()
	for index, project := range filtered {
		if filepath.Clean(project.Path) == filepath.Clean(selectedPath) {
			m.cursor = index
			break
		}
	}
	m.syncCursor()
}

func (m *model) setPinnedForSelected(pinned bool) tea.Cmd {
	project, ok := m.selectedProject()
	if !ok {
		m.status = "No project selected."
		return nil
	}

	cleanPath := filepath.Clean(project.Path)
	_, alreadyPinned := m.pinnedPaths[cleanPath]
	if pinned && alreadyPinned {
		m.status = fmt.Sprintf("%s is already pinned.", project.Name)
		return nil
	}
	if !pinned && !alreadyPinned {
		m.status = fmt.Sprintf("%s is not pinned.", project.Name)
		return nil
	}

	if pinned {
		m.pinnedPaths[cleanPath] = struct{}{}
	} else {
		delete(m.pinnedPaths, cleanPath)
	}

	m.refreshOrderedProjects(cleanPath)
	m.actionOpen = false

	action := fmt.Sprintf("Pinned %s to top.", project.Name)
	if !pinned {
		action = fmt.Sprintf("Removed %s from pinned projects.", project.Name)
	}
	m.status = action
	return saveStateCmd(m.stateStore, m.buildAppState(), action)
}

func (m *model) setHiddenForSelected(hidden bool) tea.Cmd {
	project, ok := m.selectedProject()
	if !ok {
		m.status = "No project selected."
		return nil
	}

	cleanPath := filepath.Clean(project.Path)
	if !hidden {
		delete(m.hiddenPaths, cleanPath)
		m.refreshOrderedProjects("")
		m.status = fmt.Sprintf("Restored %s.", project.Name)
		return saveStateCmd(m.stateStore, m.buildAppState(), m.status)
	}

	if _, alreadyHidden := m.hiddenPaths[cleanPath]; alreadyHidden {
		m.status = fmt.Sprintf("%s is already hidden.", project.Name)
		return nil
	}

	m.hiddenPaths[cleanPath] = struct{}{}
	delete(m.pinnedPaths, cleanPath)
	filteredBase := make([]domain.Project, 0, len(m.projectsBase))
	for _, candidate := range m.projectsBase {
		if filepath.Clean(candidate.Path) == cleanPath {
			continue
		}
		filteredBase = append(filteredBase, candidate)
	}
	m.projectsBase = filteredBase
	m.refreshOrderedProjects("")
	m.actionOpen = false
	m.status = fmt.Sprintf("Hid %s from RepoDock.", project.Name)
	return saveStateCmd(m.stateStore, m.buildAppState(), m.status)
}

func (m *model) addManualProject(path string) tea.Cmd {
	return m.addManualProjectWithName(path, "")
}

func (m *model) addManualProjectWithName(path, name string) tea.Cmd {
	path = normalizeImportedPath(path)
	if path == "" || path == "." {
		m.status = "Usage: /new /absolute/path"
		return nil
	}
	if !filepath.IsAbs(path) {
		m.status = "Manual project path must be absolute."
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		m.status = fmt.Sprintf("Project path not found: %s", path)
		return nil
	}
	if !info.IsDir() {
		m.status = "Manual project path must be a directory."
		return nil
	}

	if m.manualProjects == nil {
		m.manualProjects = make(map[string]store.ManualProject)
	}
	if _, exists := m.manualProjects[path]; exists {
		m.status = fmt.Sprintf("%s is already imported.", filepath.Base(path))
		return nil
	}

	m.manualProjects[path] = store.ManualProject{Path: path, Name: strings.TrimSpace(name)}
	m.status = fmt.Sprintf("Imported %s. Reloading projects...", filepath.Base(path))
	return tea.Batch(
		saveStateCmd(m.stateStore, m.buildAppState(), ""),
		loadProjectsCmd(m.stateStore),
	)
}

func (m *model) renameSelectedProject(name string) tea.Cmd {
	project, ok := m.selectedProject()
	if !ok {
		m.status = "No project selected."
		return nil
	}

	name = strings.TrimSpace(name)
	if name == "" {
		m.status = "Usage: /rename New Name"
		return nil
	}

	cleanPath := filepath.Clean(project.Path)
	if m.manualProjects == nil {
		m.manualProjects = make(map[string]store.ManualProject)
	}
	entry := m.manualProjects[cleanPath]
	entry.Path = cleanPath
	entry.Name = name
	m.manualProjects[cleanPath] = entry

	m.status = fmt.Sprintf("Renamed %s to %s. Reloading projects...", project.Name, name)
	return tea.Batch(
		saveStateCmd(m.stateStore, m.buildAppState(), ""),
		loadProjectsCmd(m.stateStore),
	)
}

func (m *model) clearHiddenProjects() tea.Cmd {
	if len(m.hiddenPaths) == 0 {
		m.status = "No hidden projects."
		return nil
	}

	m.hiddenPaths = make(map[string]struct{})
	m.status = "Hidden projects cleared. Reloading providers..."
	return tea.Batch(
		saveStateCmd(m.stateStore, m.buildAppState(), ""),
		loadProjectsCmd(m.stateStore),
	)
}

func (m *model) openHiddenProjects() {
	m.hiddenOpen = true
	m.actionOpen = false
	m.viewFocus = false
	m.hiddenCursor = 0
	if len(m.hiddenPaths) == 0 {
		m.status = "No hidden projects."
		return
	}
	m.status = fmt.Sprintf("Showing %d hidden projects.", len(m.hiddenPaths))
}

func (m model) hiddenProjectPaths() []string {
	if len(m.hiddenPaths) == 0 {
		return nil
	}
	paths := make([]string, 0, len(m.hiddenPaths))
	for path := range m.hiddenPaths {
		paths = append(paths, path)
	}
	sort.Slice(paths, func(i, j int) bool {
		return strings.ToLower(filepath.Base(paths[i])) < strings.ToLower(filepath.Base(paths[j]))
	})
	return paths
}

func (m *model) restoreSelectedHiddenProject() tea.Cmd {
	paths := m.hiddenProjectPaths()
	if len(paths) == 0 {
		m.status = "No hidden projects."
		return nil
	}
	if m.hiddenCursor < 0 || m.hiddenCursor >= len(paths) {
		m.hiddenCursor = 0
	}
	path := paths[m.hiddenCursor]
	delete(m.hiddenPaths, filepath.Clean(path))
	if m.hiddenCursor >= len(paths)-1 && m.hiddenCursor > 0 {
		m.hiddenCursor--
	}
	name := filepath.Base(path)
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = path
	}
	m.status = fmt.Sprintf("Restored %s. Reloading providers...", name)
	return tea.Batch(
		saveStateCmd(m.stateStore, m.buildAppState(), ""),
		loadProjectsCmd(m.stateStore),
	)
}

func (m model) projectActions() []actionEntry {
	project, ok := m.selectedProject()
	if !ok {
		return nil
	}

	pinLabel := "Pin to top"
	pinDesc := "Keep this project above the mixed list."
	pinAction := "pin"
	if m.isPinned(project.Path) {
		pinLabel = "Unpin"
		pinDesc = "Return this project to the normal list order."
		pinAction = "unpin"
	}

	actions := []actionEntry{
		{id: "shell", label: "Open Shell", desc: "Open an interactive shell in this project."},
		{id: "new", label: "Import Project", desc: "Add a manual project by drag, paste, or path input."},
	}
	if store.NewLayoutStore(project.Path).HasLayout() {
		actions = append(actions, actionEntry{id: "default-layout", label: "Load Default Layout", desc: "Open the project's default saved layout."})
	}
	if runtime.GOOS == "darwin" {
		label := "Reveal in Finder"
		desc := "Show this project in Finder."
		if m.finderRevealMode() == "open" {
			label = "Open in Finder"
			desc = "Open this project folder in Finder."
		}
		actions = append(actions, actionEntry{id: "finder", label: label, desc: desc})
	}
	actions = append(actions,
		actionEntry{id: "layout", label: "Edit Layout", desc: "Create or edit launch layouts for this project."},
		actionEntry{id: pinAction, label: pinLabel, desc: pinDesc},
		actionEntry{id: "hide", label: "Hide Project", desc: "Remove this project from RepoDock only."},
		actionEntry{id: "copy", label: "Copy Path", desc: "Copy the project path to the clipboard."},
		actionEntry{id: "sync", label: "Reload Providers", desc: "Refresh projects from detected providers."},
	)
	return actions
}

func (m *model) runSelectedAction() tea.Cmd {
	actions := m.projectActions()
	if len(actions) == 0 {
		m.status = "No actions available."
		return nil
	}
	if m.actionCursor < 0 || m.actionCursor >= len(actions) {
		m.actionCursor = 0
	}

	m.actionOpen = false
	return m.runCommand(actions[m.actionCursor].id)
}

func (m model) projectIndexAt(mouseX, mouseY int) (int, bool) {
	if m.displayMode == "list" {
		return m.listIndexAt(mouseX, mouseY)
	}

	projects := m.filteredProjects()
	if len(projects) == 0 {
		return 0, false
	}

	columns := m.gridColumns()
	maxRows := m.gridRows()
	width := m.contentWidth()
	columnWidths := gridColumnWidths(width, columns)
	cardOuterHeight := cardHeight + m.cardStyle().GetVerticalFrameSize()

	if mouseX < m.pagePaddingX() || mouseY < m.contentStartY() {
		return 0, false
	}

	relX := mouseX - m.pagePaddingX()
	relY := mouseY - m.contentStartY()
	row := relY / cardOuterHeight
	if row < 0 || row >= maxRows {
		return 0, false
	}

	col := -1
	x := 0
	for i, colWidth := range columnWidths {
		if relX >= x && relX < x+colWidth {
			col = i
			break
		}
		x += colWidth
		if i < len(columnWidths)-1 {
			if relX == x {
				return 0, false
			}
			x++
		}
	}
	if col < 0 || col >= columns {
		return 0, false
	}

	index := (m.rowOffset+row)*columns + col
	if index < 0 || index >= len(projects) {
		return 0, false
	}

	return index, true
}

func (m model) listIndexAt(mouseX, mouseY int) (int, bool) {
	projects := m.filteredProjects()
	if len(projects) == 0 {
		return 0, false
	}
	if mouseX < m.pagePaddingX() || mouseY < m.contentStartY() {
		return 0, false
	}

	row := mouseY - m.contentStartY()
	if row < 0 || row >= m.listRows() {
		return 0, false
	}

	index := m.rowOffset + row
	if index < 0 || index >= len(projects) {
		return 0, false
	}

	return index, true
}

func loadedProjectsStatus(count int, detections []sources.Detection, providerErrs []error) string {
	names := sources.AvailableProviderNames(detections)
	var base string
	if count == 0 {
		base = "No projects found from detected providers."
	} else if len(names) == 0 {
		base = fmt.Sprintf("Loaded %d projects.", count)
	} else {
		base = fmt.Sprintf("Loaded %d projects from %s.", count, strings.Join(names, ", "))
	}
	if len(providerErrs) > 0 {
		failed := make([]string, 0, len(providerErrs))
		for _, e := range providerErrs {
			failed = append(failed, e.Error())
		}
		base += fmt.Sprintf("  (%d provider error(s): %s)", len(providerErrs), strings.Join(failed, "; "))
	}
	return base
}

func demoDataset() ([]domain.Project, []sources.Detection, map[string]time.Time) {
	root := filepath.Join(os.TempDir(), "repodock-demo")
	seeds := []demoProjectSeed{
		{Name: "atlas-crm", RelPath: "client/atlas-crm", Sources: []domain.Source{domain.SourceCodex, domain.SourceClaude}, Opened: 22 * time.Minute},
		{Name: "beacon-api", RelPath: "platform/beacon-api", Sources: []domain.Source{domain.SourceCodex, domain.SourceVSCode}, Opened: 51 * time.Minute},
		{Name: "campus-portal", RelPath: "education/campus-portal", Sources: []domain.Source{domain.SourceClaude, domain.SourceCursor}, Opened: 2 * time.Hour},
		{Name: "design-system", RelPath: "frontend/design-system", Sources: []domain.Source{domain.SourceVSCode, domain.SourceCursor}, Opened: 4 * time.Hour},
		{Name: "retail-ops", RelPath: "commerce/retail-ops", Sources: []domain.Source{domain.SourceCodex}, Opened: 9 * time.Hour},
		{Name: "support-bot", RelPath: "agents/support-bot", Sources: []domain.Source{domain.SourceClaude}, Opened: 13 * time.Hour},
		{Name: "scheduler-pro", RelPath: "ops/scheduler-pro", Sources: []domain.Source{domain.SourceCodex, domain.SourceClaude, domain.SourceCursor}, Opened: 26 * time.Hour},
		{Name: "studio-site", RelPath: "marketing/studio-site", Sources: []domain.Source{domain.SourceVSCode}, Opened: 31 * time.Hour},
		{Name: "billing-core", RelPath: "finance/billing-core", Sources: []domain.Source{domain.SourceCodex, domain.SourceClaude}, Opened: 49 * time.Hour},
		{Name: "docs-hub", RelPath: "internal/docs-hub", Sources: []domain.Source{domain.SourceManual, domain.SourceVSCode}, Opened: 72 * time.Hour},
		{Name: "fleet-monitor", RelPath: "iot/fleet-monitor", Sources: []domain.Source{domain.SourceCursor, domain.SourceClaude}, Opened: 96 * time.Hour},
		{Name: "launchpad-web", RelPath: "frontend/launchpad-web", Sources: []domain.Source{domain.SourceVSCode, domain.SourceCursor}, Opened: 120 * time.Hour},
	}

	now := time.Now()
	projects := make([]domain.Project, 0, len(seeds))
	lastOpened := make(map[string]time.Time, len(seeds))
	seenProviders := make(map[string]sources.Detection)

	for _, seed := range seeds {
		path := filepath.Join(root, filepath.FromSlash(seed.RelPath))
		_ = os.MkdirAll(path, 0o755)
		projects = append(projects, domain.Project{
			Name:    seed.Name,
			Path:    path,
			Sources: append([]domain.Source(nil), seed.Sources...),
		})
		if seed.Opened > 0 {
			lastOpened[filepath.Clean(path)] = now.Add(-seed.Opened)
		}
		for _, src := range seed.Sources {
			detection, ok := demoDetectionForSource(src)
			if !ok {
				continue
			}
			seenProviders[detection.Name] = detection
		}
	}

	order := []string{"claude", "codex", "cursor", "manual", "vscode"}
	detections := make([]sources.Detection, 0, len(seenProviders))
	for _, name := range order {
		if detection, ok := seenProviders[name]; ok {
			detections = append(detections, detection)
		}
	}

	return projects, detections, lastOpened
}

func demoDetectionForSource(src domain.Source) (sources.Detection, bool) {
	name := string(src)
	switch src {
	case domain.SourceCodex, domain.SourceClaude, domain.SourceVSCode, domain.SourceCursor, domain.SourceManual:
		return sources.Detection{
			ID:        name,
			Kind:      name,
			Name:      name,
			Location:  filepath.Join(os.TempDir(), "repodock-demo"),
			Enabled:   true,
			Available: true,
			Status:    sources.StatusAvailable,
		}, true
	default:
		return sources.Detection{}, false
	}
}

func providerStatusLine(detections []sources.Detection) string {
	summary := sources.ProviderStatusSummary(detections)
	if summary == "No providers configured." {
		return summary
	}
	return fmt.Sprintf("Providers: %s.", summary)
}

func themeStatusLine(resolved uitheme.Resolved) string {
	return fmt.Sprintf("Theme: %s %s · data %s.", themeFamilyName(string(resolved.Family)), resolved.Mode, resolved.Palette.Data.Name)
}

func (m model) contentStartY() int {
	showHeader, showViewBar, _, _ := m.chromeVisibility()
	y := m.pagePaddingY()
	if showHeader {
		y += 1 + sectionGapHeight
	}
	y += m.commandBarOuterHeight() + sectionGapHeight
	if showViewBar {
		y += m.viewBarHeightFor(showViewBar) + sectionGapHeight
	}
	return y
}

func (m model) viewBarStartY() int {
	showHeader, showViewBar, _, _ := m.chromeVisibility()
	if !showViewBar {
		return -1
	}
	y := m.pagePaddingY()
	if showHeader {
		y += 1 + sectionGapHeight
	}
	y += m.commandBarOuterHeight() + sectionGapHeight
	return y
}

func projectHasSource(project domain.Project, mode string) bool {
	for _, source := range project.Sources {
		if string(source) == mode {
			return true
		}
	}
	return false
}

func (m model) availableViewModes() []string {
	modes := []string{"mixed"}
	for _, name := range sources.AvailableProviderNames(m.providers) {
		modes = append(modes, name)
	}
	return modes
}

func (m model) currentViewLabel() string {
	if strings.TrimSpace(m.viewMode) == "" {
		return "mixed"
	}
	return m.viewMode
}

func (m *model) ensureViewModeValid() {
	current := m.currentViewLabel()
	for _, mode := range m.availableViewModes() {
		if mode == current {
			return
		}
	}
	m.viewMode = "mixed"
}

func (m model) projectCountForMode(mode string) int {
	if mode == "mixed" {
		return len(m.projects)
	}

	count := 0
	for _, project := range m.projects {
		if projectHasSource(project, mode) {
			count++
		}
	}
	return count
}

func (m model) showProviderCounts() bool {
	return !m.settings.UI.HideProviderCounts
}

func (m model) viewChipLabel(mode string) string {
	if !m.showProviderCounts() {
		return mode
	}
	return fmt.Sprintf("%s %d", mode, m.projectCountForMode(mode))
}

func (m model) demoMode() bool {
	return m.settings.UI.DemoMode
}

func (m *model) setDemoMode(enabled bool) tea.Cmd {
	if m.settings.UI.DemoMode == enabled {
		if enabled {
			m.status = "Demo mode is already on."
		} else {
			m.status = "Demo mode is already off."
		}
		return nil
	}

	m.settings.UI.DemoMode = enabled
	m.input.SetValue("")
	m.cmdCursor = 0
	m.rowOffset = 0
	m.cursor = 0
	m.actionOpen = false
	m.viewFocus = false
	m.ensureViewModeValid()
	if enabled {
		m.status = "Demo mode on. Loading privacy-safe projects..."
	} else {
		m.status = "Demo mode off. Restoring real projects..."
	}

	return tea.Batch(
		saveSettingsCmd(m.settingsStore, m.settings, m.status),
		loadProjectsCmd(m.stateStore),
	)
}

func (m *model) setViewMode(mode string) bool {
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		mode = "mixed"
	}

	for _, candidate := range m.availableViewModes() {
		if candidate != mode {
			continue
		}
		m.viewMode = mode
		m.syncCursor()
		m.status = fmt.Sprintf("View mode: %s.", mode)
		return true
	}

	return false
}

func (m *model) cycleViewMode(delta int) {
	modes := m.availableViewModes()
	if len(modes) == 0 {
		return
	}

	current := m.currentViewLabel()
	index := 0
	for i, mode := range modes {
		if mode == current {
			index = i
			break
		}
	}

	index += delta
	if index < 0 {
		index = len(modes) - 1
	}
	if index >= len(modes) {
		index = 0
	}

	m.viewMode = modes[index]
	m.syncCursor()
	m.status = fmt.Sprintf("View mode: %s.", m.viewMode)
}

func (m *model) adjustSetting(delta int) {
	switch m.settingsCursor {
	case 0: // sort mode
		sorts := []string{"name", "recent"}
		idx := 0
		for i, s := range sorts {
			if s == m.sortMode {
				idx = i
				break
			}
		}
		idx = (idx + delta + len(sorts)) % len(sorts)
		m.sortMode = sorts[idx]
		m.refreshOrderedProjects("")
	case 1: // display mode
		displays := []string{"grid", "list"}
		idx := 0
		for i, d := range displays {
			if d == m.displayMode {
				idx = i
				break
			}
		}
		idx = (idx + delta + len(displays)) % len(displays)
		m.displayMode = displays[idx]
		m.syncCursor()
	case 2: // grid width
		widths := []string{"compact", "normal", "wide"}
		idx := indexOfString(widths, m.gridWidthMode())
		idx = (idx + delta + len(widths)) % len(widths)
		value := widths[idx]
		if value == "normal" {
			value = ""
		}
		m.settings.UI.GridWidth = value
		m.syncCursor()
	case 3: // show last opened
		m.settings.UI.HideProviderCounts = !m.settings.UI.HideProviderCounts
	case 4: // show last opened
		m.showLastOpened = !m.showLastOpened
	case 5: // theme family
		families := themeFamilyIDs()
		idx := indexOfString(families, m.themeFamily())
		idx = (idx + delta + len(families)) % len(families)
		_ = m.setThemeFamily(families[idx])
	case 6: // theme mode
		modes := []string{"auto", "dark", "light"}
		idx := indexOfString(modes, m.themeMode())
		idx = (idx + delta + len(modes)) % len(modes)
		_ = m.setThemeMode(modes[idx])
	case 7: // data palette
		palettes := []string{"tableau10"}
		idx := indexOfString(palettes, m.dataPalette())
		idx = (idx + delta + len(palettes)) % len(palettes)
		_ = m.setDataPalette(palettes[idx])
	case 8: // ghostty open
		opts := []string{"off", "new-window"}
		idx := indexOfString(opts, m.settings.Ghostty.Open)
		if idx < 0 {
			idx = 0
		}
		idx = (idx + delta + len(opts)) % len(opts)
		v := opts[idx]
		if v == "off" {
			v = ""
		}
		m.settings.Ghostty.Open = v
	case 9: // ghostty layout
		opts := []string{"off", "shell", "dev", "ai"}
		idx := indexOfString(opts, m.settings.Ghostty.Layout)
		if idx < 0 {
			idx = 0
		}
		idx = (idx + delta + len(opts)) % len(opts)
		v := opts[idx]
		if v == "off" {
			v = ""
		}
		m.settings.Ghostty.Layout = v
	case 10: // ghostty indicator
		m.settings.Ghostty.Indicator = !m.settings.Ghostty.Indicator
	case 11: // finder reveal mode
		opts := []string{"reveal", "open"}
		idx := indexOfString(opts, m.finderRevealMode())
		if idx < 0 {
			idx = 0
		}
		idx = (idx + delta + len(opts)) % len(opts)
		m.settings.Finder.Reveal = opts[idx]
	}
}

func (m *model) openProjectShell(project domain.Project) tea.Cmd {
	return openProjectCmd(project, m.settings)
}

func (m model) viewModeAt(mouseX, mouseY int) (string, bool) {
	_, showViewBar, _, _ := m.chromeVisibility()
	if !showViewBar {
		return "", false
	}
	startY := m.viewBarStartY()
	endY := m.contentStartY()
	if mouseY < startY || mouseY >= endY {
		return "", false
	}
	if mouseX < m.pagePaddingX() {
		return "", false
	}

	targetRow := mouseY - startY
	relX := mouseX - m.pagePaddingX()

	for _, p := range m.chipPlacements(m.contentWidth()) {
		if p.row == targetRow && relX >= p.x && relX < p.x+p.width {
			return p.mode, true
		}
		if p.row > targetRow {
			break
		}
	}
	return "", false
}

func (m *model) openThemePicker() {
	families := uitheme.Families()
	if len(families) == 0 {
		m.status = "No themes available."
		return
	}

	m.themePickerOpen = true
	m.themePickerOriginal = m.settings.Theme
	m.themePickerCursor = 0
	for i, family := range families {
		if string(family.ID) == m.themeFamily() {
			m.themePickerCursor = i
			break
		}
	}
	m.previewThemePickerSelection()
}

func (m *model) moveThemePicker(delta int) {
	families := uitheme.Families()
	if len(families) == 0 {
		return
	}

	m.themePickerCursor += delta
	if m.themePickerCursor < 0 {
		m.themePickerCursor = len(families) - 1
	}
	if m.themePickerCursor >= len(families) {
		m.themePickerCursor = 0
	}

	m.previewThemePickerSelection()
}

func (m *model) previewThemePickerSelection() {
	families := uitheme.Families()
	if len(families) == 0 {
		return
	}
	if m.themePickerCursor < 0 || m.themePickerCursor >= len(families) {
		m.themePickerCursor = 0
	}

	selected := families[m.themePickerCursor]
	m.settings.Theme.Family = string(selected.ID)
	resolved := uitheme.Resolve(m.settings.Theme)
	m.applyResolvedTheme(resolved)
	m.status = fmt.Sprintf("Previewing theme: %s.", selected.Name)
}

func (m *model) closeThemePicker(commit bool) tea.Cmd {
	if !m.themePickerOpen {
		return nil
	}

	m.themePickerOpen = false
	if !commit {
		m.settings.Theme = m.themePickerOriginal
		m.applyResolvedTheme(uitheme.Resolve(m.settings.Theme))
		m.status = "Theme preview cancelled."
		return nil
	}

	families := uitheme.Families()
	if len(families) == 0 {
		m.status = "No themes available."
		return nil
	}
	if m.themePickerCursor < 0 || m.themePickerCursor >= len(families) {
		m.themePickerCursor = 0
	}

	selected := families[m.themePickerCursor]
	m.settings.Theme.Family = string(selected.ID)
	m.applyResolvedTheme(uitheme.Resolve(m.settings.Theme))
	action := fmt.Sprintf("Theme family: %s.", selected.Name)
	m.status = action
	return saveSettingsCmd(m.settingsStore, m.settings, action)
}

func (m *model) setThemeFamily(family string) tea.Cmd {
	family = strings.TrimSpace(strings.ToLower(family))
	if family == "" {
		m.openThemePicker()
		return nil
	}
	if !isThemeFamilySupported(family) {
		m.status = fmt.Sprintf("Unknown theme family: %s", family)
		return nil
	}

	m.settings.Theme.Family = family
	resolved := uitheme.Resolve(m.settings.Theme)
	m.applyResolvedTheme(resolved)
	action := fmt.Sprintf("Theme family: %s.", themeFamilyName(family))
	m.status = action
	return saveSettingsCmd(m.settingsStore, m.settings, action)
}

func (m *model) setThemeMode(mode string) tea.Cmd {
	mode = strings.TrimSpace(strings.ToLower(mode))
	switch mode {
	case "", "auto", "dark", "light":
	default:
		m.status = fmt.Sprintf("Unknown theme mode: %s", mode)
		return nil
	}
	if mode == "" {
		mode = "auto"
	}

	m.settings.Theme.Mode = mode
	resolved := uitheme.Resolve(m.settings.Theme)
	m.applyResolvedTheme(resolved)
	action := fmt.Sprintf("Theme mode: %s.", mode)
	m.status = action
	return saveSettingsCmd(m.settingsStore, m.settings, action)
}

func (m *model) setDataPalette(name string) tea.Cmd {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		name = "tableau10"
	}
	if name != "tableau10" {
		m.status = fmt.Sprintf("Unknown data palette: %s", name)
		return nil
	}

	m.settings.Theme.DataPalette = name
	resolved := uitheme.Resolve(m.settings.Theme)
	m.applyResolvedTheme(resolved)
	action := fmt.Sprintf("Data palette: %s.", name)
	m.status = action
	return saveSettingsCmd(m.settingsStore, m.settings, action)
}

func (m *model) applyResolvedTheme(resolved uitheme.Resolved) {
	m.resolvedTheme = resolved
	applyTheme(resolved)
	applyInputStyles(&m.input)
}

func (m model) themeFamily() string {
	if strings.TrimSpace(m.settings.Theme.Family) == "" {
		return string(uitheme.FamilyTokyoNight)
	}
	return strings.TrimSpace(strings.ToLower(m.settings.Theme.Family))
}

func (m model) themeMode() string {
	if strings.TrimSpace(m.settings.Theme.Mode) == "" {
		return "auto"
	}
	return strings.TrimSpace(strings.ToLower(m.settings.Theme.Mode))
}

func (m model) dataPalette() string {
	if strings.TrimSpace(m.settings.Theme.DataPalette) == "" {
		return "tableau10"
	}
	return strings.TrimSpace(strings.ToLower(m.settings.Theme.DataPalette))
}

func boolToOnOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}

func (m model) gridWidthMode() string {
	switch strings.TrimSpace(strings.ToLower(m.settings.UI.GridWidth)) {
	case "compact", "wide":
		return strings.TrimSpace(strings.ToLower(m.settings.UI.GridWidth))
	default:
		return "normal"
	}
}

func (m model) finderRevealMode() string {
	switch strings.TrimSpace(strings.ToLower(m.settings.Finder.Reveal)) {
	case "open":
		return "open"
	default:
		return "reveal"
	}
}

func indexOfString(values []string, current string) int {
	for i, value := range values {
		if value == current {
			return i
		}
	}
	return 0
}

func themeFamilyIDs() []string {
	metas := uitheme.Families()
	ids := make([]string, 0, len(metas))
	for _, meta := range metas {
		ids = append(ids, string(meta.ID))
	}
	return ids
}

func isThemeFamilySupported(family string) bool {
	for _, candidate := range themeFamilyIDs() {
		if candidate == family {
			return true
		}
	}
	return false
}

func themeFamilyName(family string) string {
	for _, meta := range uitheme.Families() {
		if string(meta.ID) == family {
			return meta.Name
		}
	}
	return family
}

func renderThemePreviewStrip(p uitheme.Palette, width int) string {
	dataColor := p.AccentAlt
	if len(p.Data.Categorical) > 0 {
		dataColor = p.Data.Categorical[0]
	}

	segments := []struct {
		label string
		color string
	}{
		{label: "surface", color: p.Surface},
		{label: "accent", color: p.Accent},
		{label: "alt", color: p.AccentAlt},
		{label: "data", color: dataColor},
	}

	swatches := make([]string, 0, len(segments))
	for _, segment := range segments {
		block := lipgloss.NewStyle().
			Background(lipgloss.Color(segment.color)).
			Foreground(lipgloss.Color(segment.color)).
			Render("   ")
		label := currentStyles.subtle.Render(segment.label)
		swatches = append(swatches, lipgloss.JoinHorizontal(lipgloss.Left, block, " ", label))
	}

	return renderSingleLine("  ", lipgloss.JoinHorizontal(lipgloss.Left, swatches...), width, lipgloss.NewStyle())
}

func dataColorForViewMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "codex":
		return paletteColor(0)
	case "claude":
		return paletteColor(1)
	case "vscode":
		return paletteColor(2)
	case "cursor":
		return paletteColor(3)
	case "manual":
		return paletteColor(4)
	default:
		return ""
	}
}

func primarySourceColor(sources []domain.Source) string {
	if len(sources) == 0 {
		return activeTheme.Palette.AccentAlt
	}
	return dataColorForViewMode(string(sources[0]))
}

func paletteColor(index int) string {
	colors := activeTheme.Palette.Data.Categorical
	if len(colors) == 0 {
		return activeTheme.Palette.AccentAlt
	}
	return colors[index%len(colors)]
}

// ── Layout editor ────────────────────────────────────────────────────────────

func (m *model) openLayoutEditor(project domain.Project) {
	m.layoutProject = project
	m.layoutOpen = true
	m.layoutDetailOpen = false
	m.layoutNaming = false
	m.layoutDeleteConfirm = false
	m.layoutNameInput.Blur()
	m.layoutNameInput.SetValue("")
	m.layoutEditing = false
	m.layoutField = 0
	m.reloadLayouts("")
	m.status = fmt.Sprintf("Layout summary: %s.", project.Name)
}

func (m *model) layoutLoadSaved() {
	if len(m.layouts) == 0 {
		m.status = fmt.Sprintf("No saved layouts for %s.", m.layoutProject.Name)
		return
	}
	m.loadSelectedLayout()
	m.layoutDetailOpen = false
	m.status = fmt.Sprintf("Loaded layout %s for %s.", m.layoutName, m.layoutProject.Name)
}

func (m *model) reloadLayouts(preferName string) {
	ls := store.NewLayoutStore(m.layoutProject.Path)
	collection, err := ls.LoadCollection()
	if err != nil {
		m.layouts = nil
		m.layoutDefault = ""
		m.layoutSelected = 0
		m.layoutName = ""
		m.layoutPanes = []store.Pane{{Command: ""}}
		m.layoutHasSaved = false
		m.layoutDirty = false
		m.layoutSource = "default"
		m.status = fmt.Sprintf("Layout load failed: %v", err)
		return
	}

	m.layouts = append([]store.Layout(nil), collection.Layouts...)
	m.layoutDefault = collection.Default
	m.layoutCursor = 0
	m.layoutField = 0
	m.layoutEditing = false
	m.layoutDeleteConfirm = false
	m.layoutInput.Blur()

	if len(m.layouts) == 0 {
		m.layoutSelected = 0
		m.layoutDefault = ""
		m.layoutName = ""
		m.layoutPanes = []store.Pane{{Command: ""}}
		m.layoutDirty = false
		m.layoutSource = "default"
		return
	}

	selectedName := preferName
	if selectedName == "" {
		selectedName = collection.Default
	}
	idx := indexOfLayout(m.layouts, selectedName)
	if idx < 0 {
		idx = 0
	}
	m.layoutSelected = idx
	m.loadSelectedLayout()
}

func (m *model) moveLayoutSelection(delta int) {
	if len(m.layouts) == 0 {
		return
	}
	m.layoutSelected += delta
	if m.layoutSelected < 0 {
		m.layoutSelected = len(m.layouts) - 1
	}
	if m.layoutSelected >= len(m.layouts) {
		m.layoutSelected = 0
	}
	m.loadSelectedLayout()
}

func (m *model) loadSelectedLayout() {
	if len(m.layouts) == 0 {
		m.layoutDefault = ""
		m.layoutName = ""
		m.layoutPanes = []store.Pane{{Command: ""}}
		m.layoutHasSaved = false
		m.layoutDirty = false
		m.layoutSource = "default"
		return
	}
	selected := m.layouts[m.layoutSelected]
	m.layoutName = selected.Name
	m.layoutPanes = append([]store.Pane(nil), selected.Panes...)
	m.layoutHasSaved = true
	m.layoutDirty = false
	m.layoutSource = "saved"
	m.layoutCursor = 0
	m.layoutField = 0
	m.layoutEditing = false
	m.layoutInput.Blur()
}

func (m *model) beginNewLayout() {
	m.layoutNaming = true
	m.layoutDetailOpen = false
	m.layoutEditing = false
	m.layoutNameInput.SetValue("")
	m.layoutNameInput.Focus()
	m.status = "Enter a new layout name."
}

func (m *model) finishNewLayout() tea.Cmd {
	name := strings.TrimSpace(m.layoutNameInput.Value())
	if name == "" {
		m.status = "Layout name can't be empty."
		return nil
	}
	normalized := strings.ToLower(strings.Join(strings.Fields(name), "-"))
	if indexOfLayout(m.layouts, normalized) >= 0 {
		m.status = fmt.Sprintf("Layout %s already exists.", normalized)
		return nil
	}

	m.layoutNaming = false
	m.layoutNameInput.Blur()
	m.layoutNameInput.SetValue("")
	m.layoutName = normalized
	m.layoutPanes = []store.Pane{{Command: ""}}
	m.layoutCursor = 0
	m.layoutField = 0
	m.layoutDirty = true
	m.layoutSource = "edited"
	m.layoutHasSaved = false
	m.layoutDetailOpen = true
	m.status = fmt.Sprintf("Created new layout %s.", normalized)
	return m.autosaveLayoutCmd()
}

func indexOfLayout(layouts []store.Layout, name string) int {
	target := strings.TrimSpace(strings.ToLower(name))
	for i, layout := range layouts {
		if strings.TrimSpace(strings.ToLower(layout.Name)) == target {
			return i
		}
	}
	return -1
}

func (m model) autosaveLayoutCmd() tea.Cmd {
	if strings.TrimSpace(m.layoutName) == "" {
		return nil
	}
	return saveLayoutCmd(m.layoutProject, m.layoutName, m.layoutPanes, true)
}

func (m *model) layoutStartEdit() {
	if len(m.layoutPanes) == 0 {
		return
	}
	m.layoutField = 0
	m.layoutInput.SetValue(m.layoutPanes[m.layoutCursor].Command)
	m.layoutInput.Focus()
	m.layoutEditing = true
}

// layoutAdjust cycles field values with left/right keys.
func (m *model) layoutAdjust(delta int) {
	if len(m.layoutPanes) == 0 || m.layoutCursor >= len(m.layoutPanes) {
		return
	}
	p := &m.layoutPanes[m.layoutCursor]
	if m.layoutCursor == 0 {
		// First pane: nothing to cycle
		return
	}
	switch m.layoutField {
	case 1: // direction
		if p.Direction == "right" || p.Direction == "" {
			p.Direction = "down"
		} else {
			p.Direction = "right"
		}
		m.layoutDirty = true
		m.layoutSource = "edited"
	case 2: // splitFrom
		max := m.layoutCursor - 1
		p.SplitFrom = (p.SplitFrom + delta + max + 1) % (max + 1)
		m.layoutDirty = true
		m.layoutSource = "edited"
	default: // field 0 — cycle field instead
		if delta > 0 {
			m.layoutField = 1
		}
	}
}

func (m *model) layoutSplitSelected(direction string) {
	if len(m.layoutPanes) == 0 {
		m.layoutPanes = []store.Pane{{Command: ""}}
		m.layoutCursor = 0
		m.layoutDirty = true
		m.layoutSource = "edited"
		return
	}
	if direction != "down" {
		direction = "right"
	}
	from := m.layoutCursor
	if from < 0 || from >= len(m.layoutPanes) {
		from = len(m.layoutPanes) - 1
	}
	m.layoutPanes = append(m.layoutPanes, store.Pane{
		Command:   "",
		SplitFrom: from,
		Direction: direction,
	})
	m.layoutCursor = len(m.layoutPanes) - 1
	m.layoutField = 0
	m.layoutDirty = true
	m.layoutSource = "edited"
}

func (m *model) layoutSelectParent() {
	if len(m.layoutPanes) == 0 || m.layoutCursor <= 0 || m.layoutCursor >= len(m.layoutPanes) {
		return
	}
	parent := m.layoutPanes[m.layoutCursor].SplitFrom
	if parent >= 0 && parent < m.layoutCursor {
		m.layoutCursor = parent
		m.layoutField = 0
	}
}

func (m *model) layoutSelectFirstChild() {
	for i := 1; i < len(m.layoutPanes); i++ {
		if m.layoutPanes[i].SplitFrom == m.layoutCursor {
			m.layoutCursor = i
			m.layoutField = 0
			return
		}
	}
}

func (m model) layoutCanvasWidth() int {
	panelWidth := m.contentWidth()
	if panelWidth >= 28 {
		panelWidth -= 4
	}
	panelStyle := currentStyles.panel.BorderForeground(lipgloss.Color(activeTheme.Palette.BorderActive))
	bodyWidth := max(1, panelWidth-panelStyle.GetHorizontalFrameSize())
	if bodyWidth < 52 {
		return max(24, bodyWidth)
	}
	return max(24, max(20, (bodyWidth*2)/3))
}

func layoutCanvasHeight(paneCount int) int {
	return max(12, min(22, 10+paneCount*2))
}

func rectCenter(rect layoutRect) (int, int) {
	return rect.x + rect.w/2, rect.y + rect.h/2
}

func axisOverlap(aStart, aLen, bStart, bLen int) int {
	aEnd := aStart + aLen
	bEnd := bStart + bLen
	start := max(aStart, bStart)
	end := min(aEnd, bEnd)
	if end <= start {
		return 0
	}
	return end - start
}

func (m *model) layoutMoveSelection(direction string) {
	if len(m.layoutPanes) == 0 || m.layoutCursor < 0 || m.layoutCursor >= len(m.layoutPanes) {
		return
	}

	rects := computeLayoutRects(m.layoutPanes, m.layoutCanvasWidth(), layoutCanvasHeight(len(m.layoutPanes)))
	current := rects[m.layoutCursor]
	curCX, curCY := rectCenter(current)

	best := -1
	bestScore := int(^uint(0) >> 1)

	for i, candidate := range rects {
		if i == m.layoutCursor {
			continue
		}

		candCX, candCY := rectCenter(candidate)
		overlap := 0
		primary := 0
		secondary := 0
		valid := false

		switch direction {
		case "up":
			if candidate.y >= current.y {
				continue
			}
			valid = true
			overlap = axisOverlap(current.x, current.w, candidate.x, candidate.w)
			primary = current.y - (candidate.y + candidate.h)
			if primary < 0 {
				primary = 0
			}
			secondary = abs(curCX - candCX)
		case "down":
			if candidate.y+candidate.h <= current.y+current.h {
				continue
			}
			valid = true
			overlap = axisOverlap(current.x, current.w, candidate.x, candidate.w)
			primary = candidate.y - (current.y + current.h)
			if primary < 0 {
				primary = 0
			}
			secondary = abs(curCX - candCX)
		case "left":
			if candidate.x >= current.x {
				continue
			}
			valid = true
			overlap = axisOverlap(current.y, current.h, candidate.y, candidate.h)
			primary = current.x - (candidate.x + candidate.w)
			if primary < 0 {
				primary = 0
			}
			secondary = abs(curCY - candCY)
		case "right":
			if candidate.x+candidate.w <= current.x+current.w {
				continue
			}
			valid = true
			overlap = axisOverlap(current.y, current.h, candidate.y, candidate.h)
			primary = candidate.x - (current.x + current.w)
			if primary < 0 {
				primary = 0
			}
			secondary = abs(curCY - candCY)
		}

		if !valid {
			continue
		}

		score := primary*1000 + secondary
		if overlap == 0 {
			score += 100000
		} else {
			score -= min(overlap, 999)
		}

		if score < bestScore {
			bestScore = score
			best = i
		}
	}

	if best >= 0 {
		m.layoutCursor = best
		m.layoutField = 0
	}
}

func (m *model) layoutToggleDirection() {
	if len(m.layoutPanes) == 0 || m.layoutCursor <= 0 || m.layoutCursor >= len(m.layoutPanes) {
		return
	}
	if m.layoutPanes[m.layoutCursor].Direction == "down" {
		m.layoutPanes[m.layoutCursor].Direction = "right"
	} else {
		m.layoutPanes[m.layoutCursor].Direction = "down"
	}
	m.layoutDirty = true
	m.layoutSource = "edited"
}

func (m *model) layoutDeletePane() {
	if len(m.layoutPanes) <= 1 {
		return
	}
	i := m.layoutCursor
	deletedParent := 0
	if i > 0 {
		deletedParent = m.layoutPanes[i].SplitFrom
	}
	m.layoutPanes = append(m.layoutPanes[:i], m.layoutPanes[i+1:]...)

	// Remap SplitFrom references after compaction:
	//   - was pointing at the deleted pane → inherit deleted pane's own parent
	//   - was pointing at an index > i     → shift down by 1
	//   - pane 0 never has a SplitFrom     → skip
	for j := range m.layoutPanes {
		if j == 0 {
			continue
		}
		sf := m.layoutPanes[j].SplitFrom
		switch {
		case sf == i:
			// parent was deleted — attach to grandparent, clamped to valid range
			gp := deletedParent
			if gp >= j {
				gp = j - 1
			}
			m.layoutPanes[j].SplitFrom = gp
		case sf > i:
			m.layoutPanes[j].SplitFrom = sf - 1
		}
		// Sanity-clamp: SplitFrom must always be < j
		if m.layoutPanes[j].SplitFrom >= j {
			m.layoutPanes[j].SplitFrom = j - 1
		}
	}

	if m.layoutCursor >= len(m.layoutPanes) {
		m.layoutCursor = len(m.layoutPanes) - 1
	}
	m.layoutDirty = true
	m.layoutSource = "edited"
}

func grabLayoutFromGhosttyCmd() tea.Cmd {
	return func() tea.Msg {
		names := ghostty.ReadCurrentPanes()
		if len(names) == 0 {
			return layoutGrabbedMsg{panes: []store.Pane{{Command: ""}}}
		}
		panes := make([]store.Pane, len(names))
		for i, name := range names {
			cmd := inferCommand(name)
			from := 0
			if i > 0 {
				from = i - 1
			}
			panes[i] = store.Pane{
				Command:   cmd,
				SplitFrom: from,
				Direction: "right",
			}
		}
		return layoutGrabbedMsg{panes: panes}
	}
}

// inferCommand maps a Ghostty terminal process name to the command that launched it.
func inferCommand(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "claude"):
		return "claude"
	case strings.Contains(lower, "codex"):
		return "codex"
	default:
		return ""
	}
}

func saveLayoutCmd(project domain.Project, name string, panes []store.Pane, auto bool) tea.Cmd {
	return func() tea.Msg {
		ls := store.NewLayoutStore(project.Path)
		collection, err := ls.LoadCollection()
		if err != nil {
			return layoutSavedMsg{project: project.Name, name: name, auto: auto, err: err}
		}

		layout := store.Layout{Name: name, Panes: panes}
		if idx := indexOfLayout(collection.Layouts, name); idx >= 0 {
			collection.Layouts[idx] = layout
		} else {
			collection.Layouts = append(collection.Layouts, layout)
		}
		if collection.Default == "" {
			collection.Default = name
		}
		err = ls.SaveCollection(collection)
		return layoutSavedMsg{project: project.Name, name: name, auto: auto, err: err}
	}
}

func deleteLayoutCmd(project domain.Project, name string) tea.Cmd {
	return func() tea.Msg {
		ls := store.NewLayoutStore(project.Path)
		collection, err := ls.LoadCollection()
		if err != nil {
			return layoutDeletedMsg{project: project.Name, name: name, err: err}
		}

		idx := indexOfLayout(collection.Layouts, name)
		if idx < 0 {
			return layoutDeletedMsg{project: project.Name, name: name, err: fmt.Errorf("layout not found")}
		}

		collection.Layouts = append(collection.Layouts[:idx], collection.Layouts[idx+1:]...)
		if len(collection.Layouts) == 0 {
			err = ls.Delete()
			return layoutDeletedMsg{project: project.Name, name: name, err: err}
		}

		if strings.TrimSpace(strings.ToLower(collection.Default)) == strings.TrimSpace(strings.ToLower(name)) {
			collection.Default = collection.Layouts[0].Name
		}
		err = ls.SaveCollection(collection)
		return layoutDeletedMsg{project: project.Name, name: name, err: err}
	}
}

func setDefaultLayoutCmd(project domain.Project, name string) tea.Cmd {
	return func() tea.Msg {
		ls := store.NewLayoutStore(project.Path)
		collection, err := ls.LoadCollection()
		if err != nil {
			return layoutDefaultSetMsg{project: project.Name, name: name, err: err}
		}
		if _, ok := collection.Find(name); !ok {
			return layoutDefaultSetMsg{project: project.Name, name: name, err: fmt.Errorf("layout not found")}
		}
		collection.Default = name
		err = ls.SaveCollection(collection)
		return layoutDefaultSetMsg{project: project.Name, name: name, err: err}
	}
}

func applyLayoutCmd(project domain.Project, panes []store.Pane) tea.Cmd {
	return func() tea.Msg {
		err := ghostty.OpenFromLayout(project.Path, panes)
		return layoutAppliedMsg{project: project.Name, err: err}
	}
}

func renderLayoutPanel(m model, width int) string {
	if !m.layoutDetailOpen {
		return renderLayoutSummaryPanel(m, width)
	}
	return renderLayoutDetailPanel(m, width)
}

func renderLayoutSummaryPanel(m model, width int) string {
	subtle := currentStyles.subtle
	lines := []string{
		"  " + currentStyles.panelTitle.Render("Layout") + "  " + subtle.Render(m.layoutProject.Name),
		"  " + subtle.Render(fmt.Sprintf("layouts: %d  ·  current: %s  ·  default: %s  ·  state: %s", len(m.layouts), layoutDisplayName(m.layoutName), layoutDisplayName(m.layoutDefault), layoutSourceLabel(m.layoutSource, m.layoutDirty))),
		"",
	}

	if m.layoutNaming {
		lines = append(lines, "  "+subtle.Render("New layout"))
		lines = append(lines, "  "+m.layoutNameInput.View())
		lines = append(lines, "")
		lines = append(lines, "  "+subtle.Render("Press enter to create a new named layout."))
	} else if len(m.layouts) == 0 {
		lines = append(lines, "  "+currentStyles.listNormal.Render("No saved layouts yet."))
		lines = append(lines, "  "+subtle.Render("Create one with [n], or grab current Ghostty panes then save."))
	} else {
		for i, layout := range m.layouts {
			prefix := "  "
			nameStyle := currentStyles.listNormal
			if i == m.layoutSelected {
				prefix = currentStyles.panelSelected.Render("▶ ")
				nameStyle = currentStyles.listSelected
			}
			label := layout.Name
			if label == "" {
				label = "default"
			}
			marker := ""
			if strings.EqualFold(layout.Name, m.layoutDefault) {
				marker = subtle.Render(" [default]")
			}
			lines = append(lines, prefix+nameStyle.Render(label)+marker+"  "+subtle.Render(fmt.Sprintf("%d panes", len(layout.Panes))))
		}
	}
	lines = append(lines, "")
	if m.layoutDeleteConfirm {
		lines = append(lines, "  "+currentStyles.panelSelected.Render("Delete "+layoutDisplayName(m.layoutName)+"? This cannot be undone."))
		lines = append(lines, "  "+subtle.Render("Press Enter to confirm or Esc to cancel."))
		lines = append(lines, "")
	}
	lines = append(lines, "  "+subtle.Render("Apply auto-submits each pane command with Enter."))
	lines = append(lines, "  "+subtle.Render("[n] new  [f] set default  [x] delete  [enter] inspect/edit  [l] load selected  [g] grab current  [s] save current  [p] apply current"))

	panelWidth := width
	if width >= 28 {
		panelWidth = width - 4
	}
	panel := fitOuterWidth(
		currentStyles.panel.BorderForeground(lipgloss.Color(activeTheme.Palette.BorderActive)),
		panelWidth,
	)
	return panel.Render(strings.Join(lines, "\n"))
}

func renderLayoutDetailPanel(m model, width int) string {
	subtle := currentStyles.subtle

	lines := []string{
		"  " + currentStyles.panelTitle.Render("Layout Details") + "  " + subtle.Render(m.layoutProject.Name+" / "+layoutDisplayName(m.layoutName)),
		"  " + subtle.Render(fmt.Sprintf("saved: %s  ·  state: %s  ·  panes: %d", yesNo(m.layoutHasSaved), layoutSourceLabel(m.layoutSource, m.layoutDirty), len(m.layoutPanes))),
		"",
	}

	panelWidth := width
	if width >= 28 {
		panelWidth = width - 4
	}
	panelStyle := currentStyles.panel.BorderForeground(lipgloss.Color(activeTheme.Palette.BorderActive))
	bodyWidth := max(1, panelWidth-panelStyle.GetHorizontalFrameSize())
	if bodyWidth < 52 {
		lines = append(lines, renderLayoutCanvas(m.layoutPanes, m.layoutCursor, bodyWidth))
		lines = append(lines, "")
		lines = append(lines, renderLayoutInspector(m, bodyWidth))
	} else {
		previewWidth := max(20, (bodyWidth*2)/3)
		inspectorWidth := max(18, bodyWidth-previewWidth-2)
		preview := renderLayoutCanvas(m.layoutPanes, m.layoutCursor, previewWidth)
		inspector := renderLayoutInspector(m, inspectorWidth)
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, preview, "  ", inspector))
	}
	lines = append(lines, "")
	lines = append(lines, "  "+subtle.Render("Apply auto-submits each pane command with Enter."))
	lines = append(lines, "  "+subtle.Render("[r] split right  [b] split down  [t] toggle dir  [e] edit cmd  [d] del  [g] grab  [s] save  [p] apply"))

	panel := fitOuterWidth(panelStyle, panelWidth)
	return panel.Render(strings.Join(lines, "\n"))
}

func layoutSourceLabel(source string, dirty bool) string {
	base := "default"
	switch source {
	case "saved":
		base = "saved"
	case "ghostty":
		base = "grabbed"
	case "edited":
		base = "edited"
	}
	if dirty {
		return base + " (unsaved)"
	}
	return base
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func layoutDisplayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "(unsaved)"
	}
	return name
}

func renderLayoutPreview(panes []store.Pane, selected int) string {
	if len(panes) == 0 {
		return ""
	}

	children := make(map[int][]int, len(panes))
	for i := 1; i < len(panes); i++ {
		parent := panes[i].SplitFrom
		if parent < 0 || parent >= i {
			parent = i - 1
		}
		children[parent] = append(children[parent], i)
	}

	lines := []string{previewPaneLine(0, panes[0], selected == 0)}
	lines = append(lines, previewChildren(children, panes, 0, "", selected)...)
	return strings.Join(lines, "\n")
}

func renderLayoutCanvas(panes []store.Pane, selected int, width int) string {
	if len(panes) == 0 {
		return lipgloss.NewStyle().Width(width).Render(currentStyles.subtle.Render("No panes yet."))
	}
	canvasWidth := max(24, width)
	canvasHeight := max(12, min(22, 10+len(panes)*2))
	rects := computeLayoutRects(panes, canvasWidth, canvasHeight)
	grid := make([][]rune, canvasHeight)
	for y := range grid {
		grid[y] = make([]rune, canvasWidth)
		for x := range grid[y] {
			grid[y][x] = ' '
		}
	}

	for i, rect := range rects {
		drawPaneRect(grid, rect, panes[i], i, i == selected)
	}

	lines := make([]string, len(grid))
	for i := range grid {
		lines[i] = strings.TrimRight(string(grid[i]), " ")
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

type layoutRect struct {
	x int
	y int
	w int
	h int
}

func computeLayoutRects(panes []store.Pane, width, height int) []layoutRect {
	rects := make([]layoutRect, len(panes))
	if len(panes) == 0 {
		return rects
	}
	rects[0] = layoutRect{x: 0, y: 0, w: width, h: height}
	for i := 1; i < len(panes); i++ {
		parent := panes[i].SplitFrom
		if parent < 0 || parent >= i {
			parent = i - 1
		}
		parentRect := rects[parent]
		if panes[i].Direction == "down" {
			topH := max(4, parentRect.h/2)
			if topH >= parentRect.h {
				topH = max(3, parentRect.h-3)
			}
			bottomH := max(3, parentRect.h-topH)
			rects[parent] = layoutRect{x: parentRect.x, y: parentRect.y, w: parentRect.w, h: topH}
			rects[i] = layoutRect{x: parentRect.x, y: parentRect.y + topH, w: parentRect.w, h: bottomH}
		} else {
			leftW := max(8, parentRect.w/2)
			if leftW >= parentRect.w {
				leftW = max(6, parentRect.w-6)
			}
			rightW := max(6, parentRect.w-leftW)
			rects[parent] = layoutRect{x: parentRect.x, y: parentRect.y, w: leftW, h: parentRect.h}
			rects[i] = layoutRect{x: parentRect.x + leftW, y: parentRect.y, w: rightW, h: parentRect.h}
		}
	}
	return rects
}

func drawPaneRect(grid [][]rune, rect layoutRect, pane store.Pane, index int, selected bool) {
	if rect.w < 3 || rect.h < 3 {
		return
	}
	h := len(grid)
	w := len(grid[0])
	x0 := max(0, rect.x)
	y0 := max(0, rect.y)
	x1 := min(w-1, rect.x+rect.w-1)
	y1 := min(h-1, rect.y+rect.h-1)
	if x1 <= x0 || y1 <= y0 {
		return
	}

	hLine := '─'
	vLine := '│'
	topLeft, topRight, bottomLeft, bottomRight := '┌', '┐', '└', '┘'
	if selected {
		hLine = '━'
		vLine = '┃'
		topLeft, topRight, bottomLeft, bottomRight = '┏', '┓', '┗', '┛'
	}

	for x := x0; x <= x1; x++ {
		grid[y0][x] = hLine
		grid[y1][x] = hLine
	}
	for y := y0; y <= y1; y++ {
		grid[y][x0] = vLine
		grid[y][x1] = vLine
	}
	grid[y0][x0], grid[y0][x1], grid[y1][x0], grid[y1][x1] = topLeft, topRight, bottomLeft, bottomRight

	cmd := strings.TrimSpace(pane.Command)
	if cmd == "" {
		cmd = "shell"
	}
	labelLines := []string{fmt.Sprintf("%d", index+1)}
	if rect.w >= 8 {
		labelLines = append(labelLines, truncate(cmd, max(1, rect.w-2)))
	}
	if index > 0 && rect.h >= 6 && rect.w >= 10 {
		dir := pane.Direction
		if dir == "" {
			dir = "right"
		}
		labelLines = append(labelLines, truncate(dir, max(1, rect.w-2)))
	}
	for i, line := range labelLines {
		yy := y0 + 1 + i
		if yy >= y1 {
			break
		}
		for x, r := range []rune(line) {
			xx := x0 + 1 + x
			if xx >= x1 {
				break
			}
			grid[yy][xx] = r
		}
	}
}

func renderLayoutInspector(m model, width int) string {
	subtle := currentStyles.subtle
	if len(m.layoutPanes) == 0 || m.layoutCursor >= len(m.layoutPanes) {
		return lipgloss.NewStyle().Width(width).Render(subtle.Render("No pane selected."))
	}

	pane := m.layoutPanes[m.layoutCursor]
	cmd := strings.TrimSpace(pane.Command)
	if cmd == "" {
		cmd = "shell"
	}
	dir := pane.Direction
	if dir == "" {
		dir = "root"
	}
	parent := "none"
	if m.layoutCursor > 0 {
		parent = fmt.Sprintf("pane %d", pane.SplitFrom+1)
	}

	lines := []string{
		currentStyles.panelTitle.Render("Selected Pane"),
		currentStyles.listSelected.Render(fmt.Sprintf("pane %d", m.layoutCursor+1)),
		"",
		subtle.Render("command"),
		currentStyles.listNormal.Render(cmd),
		"",
		subtle.Render("direction"),
		currentStyles.listNormal.Render(dir),
		"",
		subtle.Render("split from"),
		currentStyles.listNormal.Render(parent),
		"",
		subtle.Render("actions"),
		currentStyles.listNormal.Render("r  split right"),
		currentStyles.listNormal.Render("b  split down"),
		currentStyles.listNormal.Render("t  toggle direction"),
		currentStyles.listNormal.Render("e  edit command"),
		currentStyles.listNormal.Render("d  delete pane"),
	}

	if m.layoutEditing {
		lines = append(lines, "")
		lines = append(lines, subtle.Render("edit command"))
		lines = append(lines, m.layoutInput.View())
	}

	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

func previewChildren(children map[int][]int, panes []store.Pane, parent int, prefix string, selected int) []string {
	var lines []string
	kids := children[parent]
	for i, idx := range kids {
		last := i == len(kids)-1
		connector := "|- "
		nextPrefix := prefix + "|  "
		if last {
			connector = "`- "
			nextPrefix = prefix + "   "
		}

		dir := panes[idx].Direction
		if dir == "" {
			dir = "right"
		}
		line := prefix + connector + dir + " -> " + previewPaneLine(idx, panes[idx], idx == selected)
		lines = append(lines, line)
		lines = append(lines, previewChildren(children, panes, idx, nextPrefix, selected)...)
	}
	return lines
}

func previewPaneLine(index int, pane store.Pane, selected bool) string {
	label := strings.TrimSpace(pane.Command)
	if label == "" {
		label = "shell"
	}
	if selected {
		return fmt.Sprintf("[%d] %s", index+1, label)
	}
	return fmt.Sprintf("%d %s", index+1, label)
}

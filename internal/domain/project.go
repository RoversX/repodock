package domain

type Source string

const (
	SourceManual      Source = "manual"
	SourceCodex       Source = "codex"
	SourceClaude      Source = "claude"
	SourceVSCode      Source = "vscode"
	SourceCursor      Source = "cursor"
	SourceAntigravity Source = "antigravity"
	SourcePi          Source = "pi"
	SourceOpenCode    Source = "opencode"
)

type Project struct {
	Name    string
	Path    string
	Sources []Source
}

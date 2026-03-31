package actions

type Action struct {
	Key     string
	Label   string
	Command string
}

var DefaultActions = []Action{
	{Key: "enter", Label: "Open shell in project", Command: "shell"},
	{Key: "codex", Label: "Launch Codex in project", Command: "codex"},
	{Key: "claude", Label: "Launch Claude in project", Command: "claude"},
	{Key: "code", Label: "Open in VS Code", Command: "code ."},
	{Key: "cursor", Label: "Open in Cursor", Command: "cursor ."},
}

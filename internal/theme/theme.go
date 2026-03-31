package theme

import (
	"os/exec"
	"runtime"
	"strings"

	"github.com/roversx/repodock/internal/store"
)

type Mode string

const (
	ModeAuto  Mode = "auto"
	ModeDark  Mode = "dark"
	ModeLight Mode = "light"
)

type Family string

const (
	FamilyTokyoNight Family = "tokyonight"
	FamilyCatppuccin Family = "catppuccin"
	FamilyNord       Family = "nord"
)

type DataPaletteName string

const (
	DataPaletteTableau10 DataPaletteName = "tableau10"
)

type DataPalette struct {
	Name        DataPaletteName
	Categorical []string
}

type Palette struct {
	Background     string
	Surface        string
	SurfaceAlt     string
	Overlay        string
	Border         string
	BorderActive   string
	Accent         string
	AccentAlt      string
	Text           string
	TextMuted      string
	TextSubtle     string
	TextOnAccent   string
	Placeholder    string
	Selection      string
	SelectionMuted string
	Positive       string
	Negative       string
	Warning        string
	Info           string
	Data           DataPalette
}

type Resolved struct {
	Family  Family
	Mode    Mode
	Theme   string
	Palette Palette
}

type FamilyMeta struct {
	ID          Family
	Name        string
	Description string
}

func Resolve(cfg store.ThemeSettings) Resolved {
	family := parseFamily(cfg.Family)
	mode := parseMode(cfg.Mode)
	if mode == ModeAuto {
		mode = DetectSystemMode()
	}

	data := resolveDataPalette(cfg.DataPalette)
	palette := resolvePalette(family, mode, data)
	return Resolved{
		Family:  family,
		Mode:    mode,
		Theme:   string(family),
		Palette: palette,
	}
}

func Default() Resolved {
	return Resolve(store.ThemeSettings{})
}

func Families() []FamilyMeta {
	return []FamilyMeta{
		{ID: FamilyTokyoNight, Name: "Tokyo Night", Description: "Electric blue, editor-native contrast."},
		{ID: FamilyCatppuccin, Name: "Catppuccin", Description: "Soft pastel UI with warm contrast."},
		{ID: FamilyNord, Name: "Nord", Description: "Cold arctic palette with low-glare surfaces."},
	}
}

func DetectSystemMode() Mode {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output()
		if err == nil && strings.Contains(strings.ToLower(string(out)), "dark") {
			return ModeDark
		}
		return ModeLight
	}

	return ModeDark
}

func parseFamily(raw string) Family {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", "tokyonight", "tokyo-night", "tokyo_night":
		return FamilyTokyoNight
	case "catppuccin":
		return FamilyCatppuccin
	case "nord":
		return FamilyNord
	default:
		return FamilyTokyoNight
	}
}

func parseMode(raw string) Mode {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", "auto":
		return ModeAuto
	case "dark":
		return ModeDark
	case "light":
		return ModeLight
	default:
		return ModeAuto
	}
}

func resolveDataPalette(raw string) DataPalette {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", "tableau10", "tableau-10", "tableau_10":
		return DataPalette{
			Name: DataPaletteTableau10,
			Categorical: []string{
				"#4e79a7",
				"#f28e2c",
				"#e15759",
				"#76b7b2",
				"#59a14f",
				"#edc949",
				"#af7aa1",
				"#ff9da7",
				"#9c755f",
				"#bab0ab",
			},
		}
	default:
		return resolveDataPalette("")
	}
}

func resolvePalette(family Family, mode Mode, data DataPalette) Palette {
	switch family {
	case FamilyTokyoNight:
		if mode == ModeLight {
			return tokyoNightDay(data)
		}
		return tokyoNightStorm(data)
	case FamilyCatppuccin:
		if mode == ModeLight {
			return catppuccinLatte(data)
		}
		return catppuccinMocha(data)
	case FamilyNord:
		if mode == ModeLight {
			return nordLight(data)
		}
		return nordDark(data)
	default:
		return tokyoNightStorm(data)
	}
}

func tokyoNightStorm(data DataPalette) Palette {
	return Palette{
		Background:     "#24283b",
		Surface:        "#1d202f",
		SurfaceAlt:     "#2f3549",
		Overlay:        "#1f2335",
		Border:         "#414868",
		BorderActive:   "#7aa2f7",
		Accent:         "#7aa2f7",
		AccentAlt:      "#7dcfff",
		Text:           "#c0caf5",
		TextMuted:      "#a9b1d6",
		TextSubtle:     "#565f89",
		TextOnAccent:   "#24283b",
		Placeholder:    "#565f89",
		Selection:      "#c0caf5",
		SelectionMuted: "#9aa5ce",
		Positive:       "#9ece6a",
		Negative:       "#f7768e",
		Warning:        "#e0af68",
		Info:           "#7dcfff",
		Data:           data,
	}
}

func tokyoNightDay(data DataPalette) Palette {
	return Palette{
		Background:     "#e1e2e7",
		Surface:        "#d5d6db",
		SurfaceAlt:     "#c4c8da",
		Overlay:        "#dcdfe4",
		Border:         "#a1a6c5",
		BorderActive:   "#2e7de9",
		Accent:         "#2e7de9",
		AccentAlt:      "#007197",
		Text:           "#3760bf",
		TextMuted:      "#6172b0",
		TextSubtle:     "#8990b3",
		TextOnAccent:   "#ffffff",
		Placeholder:    "#8990b3",
		Selection:      "#3760bf",
		SelectionMuted: "#6172b0",
		Positive:       "#587539",
		Negative:       "#f52a65",
		Warning:        "#8c6c3e",
		Info:           "#007197",
		Data:           data,
	}
}

func catppuccinMocha(data DataPalette) Palette {
	return Palette{
		Background:     "#1e1e2e",
		Surface:        "#181825",
		SurfaceAlt:     "#313244",
		Overlay:        "#11111b",
		Border:         "#45475a",
		BorderActive:   "#89b4fa",
		Accent:         "#89b4fa",
		AccentAlt:      "#94e2d5",
		Text:           "#cdd6f4",
		TextMuted:      "#bac2de",
		TextSubtle:     "#7f849c",
		TextOnAccent:   "#11111b",
		Placeholder:    "#6c7086",
		Selection:      "#cdd6f4",
		SelectionMuted: "#bac2de",
		Positive:       "#a6e3a1",
		Negative:       "#f38ba8",
		Warning:        "#f9e2af",
		Info:           "#89dceb",
		Data:           data,
	}
}

func catppuccinLatte(data DataPalette) Palette {
	return Palette{
		Background:     "#eff1f5",
		Surface:        "#e6e9ef",
		SurfaceAlt:     "#ccd0da",
		Overlay:        "#dce0e8",
		Border:         "#bcc0cc",
		BorderActive:   "#1e66f5",
		Accent:         "#1e66f5",
		AccentAlt:      "#179299",
		Text:           "#4c4f69",
		TextMuted:      "#5c5f77",
		TextSubtle:     "#8c8fa1",
		TextOnAccent:   "#ffffff",
		Placeholder:    "#9ca0b0",
		Selection:      "#4c4f69",
		SelectionMuted: "#5c5f77",
		Positive:       "#40a02b",
		Negative:       "#d20f39",
		Warning:        "#df8e1d",
		Info:           "#209fb5",
		Data:           data,
	}
}

func nordDark(data DataPalette) Palette {
	return Palette{
		Background:     "#2E3440",
		Surface:        "#3B4252",
		SurfaceAlt:     "#434C5E",
		Overlay:        "#4C566A",
		Border:         "#4C566A",
		BorderActive:   "#88C0D0",
		Accent:         "#88C0D0",
		AccentAlt:      "#81A1C1",
		Text:           "#ECEFF4",
		TextMuted:      "#E5E9F0",
		TextSubtle:     "#D8DEE9",
		TextOnAccent:   "#2E3440",
		Placeholder:    "#D8DEE9",
		Selection:      "#ECEFF4",
		SelectionMuted: "#E5E9F0",
		Positive:       "#A3BE8C",
		Negative:       "#BF616A",
		Warning:        "#EBCB8B",
		Info:           "#8FBCBB",
		Data:           data,
	}
}

func nordLight(data DataPalette) Palette {
	return Palette{
		Background:     "#ECEFF4",
		Surface:        "#E5E9F0",
		SurfaceAlt:     "#D8DEE9",
		Overlay:        "#e1e6ee",
		Border:         "#A7B1C2",
		BorderActive:   "#5E81AC",
		Accent:         "#5E81AC",
		AccentAlt:      "#5E81AC",
		Text:           "#2E3440",
		TextMuted:      "#3B4252",
		TextSubtle:     "#4C566A",
		TextOnAccent:   "#ECEFF4",
		Placeholder:    "#4C566A",
		Selection:      "#2E3440",
		SelectionMuted: "#3B4252",
		Positive:       "#A3BE8C",
		Negative:       "#BF616A",
		Warning:        "#D08770",
		Info:           "#88C0D0",
		Data:           data,
	}
}

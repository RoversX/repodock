package theme

import (
	"testing"

	"github.com/roversx/repodock/internal/store"
)

func TestResolveLightModeTokyoNight(t *testing.T) {
	resolved := Resolve(store.ThemeSettings{
		Family:      "tokyonight",
		Mode:        "light",
		DataPalette: "tableau10",
	})

	if resolved.Mode != ModeLight {
		t.Fatalf("expected light mode, got %s", resolved.Mode)
	}
	if resolved.Family != FamilyTokyoNight {
		t.Fatalf("expected tokyonight family, got %s", resolved.Family)
	}
	if resolved.Palette.Background != "#e1e2e7" {
		t.Fatalf("unexpected light background: %s", resolved.Palette.Background)
	}
	if len(resolved.Palette.Data.Categorical) != 10 {
		t.Fatalf("expected 10 tableau colors, got %d", len(resolved.Palette.Data.Categorical))
	}
}

func TestResolveDarkModeTokyoNight(t *testing.T) {
	resolved := Resolve(store.ThemeSettings{
		Family: "tokyonight",
		Mode:   "dark",
	})

	if resolved.Mode != ModeDark {
		t.Fatalf("expected dark mode, got %s", resolved.Mode)
	}
	if resolved.Palette.Background != "#24283b" {
		t.Fatalf("unexpected dark background: %s", resolved.Palette.Background)
	}
	if resolved.Palette.Accent != "#7aa2f7" {
		t.Fatalf("unexpected dark accent: %s", resolved.Palette.Accent)
	}
}

func TestResolveKnownFamilies(t *testing.T) {
	testCases := []struct {
		name       string
		family     string
		mode       string
		wantFamily Family
		wantBG     string
		wantAccent string
	}{
		{
			name:       "catppuccin dark",
			family:     "catppuccin",
			mode:       "dark",
			wantFamily: FamilyCatppuccin,
			wantBG:     "#1e1e2e",
			wantAccent: "#89b4fa",
		},
		{
			name:       "catppuccin light",
			family:     "catppuccin",
			mode:       "light",
			wantFamily: FamilyCatppuccin,
			wantBG:     "#eff1f5",
			wantAccent: "#1e66f5",
		},
		{
			name:       "nord dark",
			family:     "nord",
			mode:       "dark",
			wantFamily: FamilyNord,
			wantBG:     "#2E3440",
			wantAccent: "#88C0D0",
		},
		{
			name:       "nord light",
			family:     "nord",
			mode:       "light",
			wantFamily: FamilyNord,
			wantBG:     "#ECEFF4",
			wantAccent: "#5E81AC",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resolved := Resolve(store.ThemeSettings{
				Family: tc.family,
				Mode:   tc.mode,
			})

			if resolved.Family != tc.wantFamily {
				t.Fatalf("expected family %s, got %s", tc.wantFamily, resolved.Family)
			}
			if resolved.Palette.Background != tc.wantBG {
				t.Fatalf("expected background %s, got %s", tc.wantBG, resolved.Palette.Background)
			}
			if resolved.Palette.Accent != tc.wantAccent {
				t.Fatalf("expected accent %s, got %s", tc.wantAccent, resolved.Palette.Accent)
			}
		})
	}
}

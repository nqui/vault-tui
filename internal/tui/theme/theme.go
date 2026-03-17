package theme

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme holds all color slots used across the app.
type Theme struct {
	Bg, Surface, Overlay, Subtle color.Color
	Text, Bright                 color.Color
	Primary, Blue, Cyan          color.Color
	Green, Yellow, Red, Orange   color.Color
	CursorBg                     color.Color
}

// Active is the currently selected theme, set once at startup.
var Active Theme

// Set selects a theme by name and sets Active. Returns an error for unknown names.
func Set(name string) error {
	switch name {
	case "tokyonight", "":
		Active = tokyonight()
	case "catppuccin":
		Active = catppuccin()
	case "gruvbox":
		Active = gruvbox()
	case "nord":
		Active = nord()
	default:
		return fmt.Errorf("unknown theme %q (valid: tokyonight, catppuccin, gruvbox, nord)", name)
	}
	return nil
}

func tokyonight() Theme {
	return Theme{
		Bg:       lipgloss.Color("#1A1B26"),
		Surface:  lipgloss.Color("#24283B"),
		Overlay:  lipgloss.Color("#414868"),
		Subtle:   lipgloss.Color("#565F89"),
		Text:     lipgloss.Color("#C0CAF5"),
		Bright:   lipgloss.Color("#E0E0FF"),
		Primary:  lipgloss.Color("#BB9AF7"),
		Blue:     lipgloss.Color("#7AA2F7"),
		Cyan:     lipgloss.Color("#7DCFFF"),
		Green:    lipgloss.Color("#9ECE6A"),
		Yellow:   lipgloss.Color("#E0AF68"),
		Red:      lipgloss.Color("#F7768E"),
		Orange:   lipgloss.Color("#FF9E64"),
		CursorBg: lipgloss.Color("#3B4261"),
	}
}

func catppuccin() Theme {
	return Theme{
		Bg:       lipgloss.Color("#1E1E2E"),
		Surface:  lipgloss.Color("#313244"),
		Overlay:  lipgloss.Color("#45475A"),
		Subtle:   lipgloss.Color("#6C7086"),
		Text:     lipgloss.Color("#CDD6F4"),
		Bright:   lipgloss.Color("#BAC2DE"),
		Primary:  lipgloss.Color("#CBA6F7"),
		Blue:     lipgloss.Color("#89B4FA"),
		Cyan:     lipgloss.Color("#94E2D5"),
		Green:    lipgloss.Color("#A6E3A1"),
		Yellow:   lipgloss.Color("#F9E2AF"),
		Red:      lipgloss.Color("#F38BA8"),
		Orange:   lipgloss.Color("#FAB387"),
		CursorBg: lipgloss.Color("#45475A"),
	}
}

func gruvbox() Theme {
	return Theme{
		Bg:       lipgloss.Color("#282828"),
		Surface:  lipgloss.Color("#3C3836"),
		Overlay:  lipgloss.Color("#504945"),
		Subtle:   lipgloss.Color("#928374"),
		Text:     lipgloss.Color("#EBDBB2"),
		Bright:   lipgloss.Color("#FBF1C7"),
		Primary:  lipgloss.Color("#D3869B"),
		Blue:     lipgloss.Color("#83A598"),
		Cyan:     lipgloss.Color("#8EC07C"),
		Green:    lipgloss.Color("#B8BB26"),
		Yellow:   lipgloss.Color("#FABD2F"),
		Red:      lipgloss.Color("#FB4934"),
		Orange:   lipgloss.Color("#FE8019"),
		CursorBg: lipgloss.Color("#504945"),
	}
}

func nord() Theme {
	return Theme{
		Bg:       lipgloss.Color("#2E3440"),
		Surface:  lipgloss.Color("#3B4252"),
		Overlay:  lipgloss.Color("#434C5E"),
		Subtle:   lipgloss.Color("#616E88"),
		Text:     lipgloss.Color("#ECEFF4"),
		Bright:   lipgloss.Color("#E5E9F0"),
		Primary:  lipgloss.Color("#B48EAD"),
		Blue:     lipgloss.Color("#81A1C1"),
		Cyan:     lipgloss.Color("#88C0D0"),
		Green:    lipgloss.Color("#A3BE8C"),
		Yellow:   lipgloss.Color("#EBCB8B"),
		Red:      lipgloss.Color("#BF616A"),
		Orange:   lipgloss.Color("#D08770"),
		CursorBg: lipgloss.Color("#434C5E"),
	}
}

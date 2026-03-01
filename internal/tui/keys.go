package tui

import (
	"charm.land/bubbles/v2/key"
)

type keyMap struct {
	Quit       key.Binding
	SwitchPane key.Binding
	Help       key.Binding
	Cancel     key.Binding

	Up       key.Binding
	Down     key.Binding
	Open     key.Binding
	Back     key.Binding
	Top      key.Binding
	Bottom   key.Binding
	Refresh  key.Binding
	New      key.Binding
	Delete   key.Binding
	Edit     key.Binding
	Versions key.Binding
	Copy     key.Binding
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	SwitchPane: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch pane"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "down"),
	),
	Open: key.NewBinding(
		key.WithKeys("enter", "l", "right"),
		key.WithHelp("enter", "open"),
	),
	Back: key.NewBinding(
		key.WithKeys("h", "left"),
		key.WithHelp("h/←", "back"),
	),
	Top: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "top"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "bottom"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new secret"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	Versions: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "versions"),
	),
	Copy: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy secret"),
	),
}

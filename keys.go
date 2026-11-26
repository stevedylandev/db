package main

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
)

var (
	keyQuit   = key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
	keyUp     = key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up"))
	keyDown   = key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down"))
	keySelect = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select"))

	keyBack    = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))
	keyEscBack = key.NewBinding(key.WithKeys("esc", "enter"), key.WithHelp("esc/enter", "back"))
	keyCancel  = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))

	keyInsert = key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "insert"))
	keyUpdate = key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update"))
	keyDelete = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete"))
	keySchema = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "schema"))

	keyColsLeft  = key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "cols left"))
	keyColsRight = key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "cols right"))

	keyFormFields = key.NewBinding(
		key.WithKeys("up", "down", "tab", "shift+tab"),
		key.WithHelp("↑/↓/tab", "fields"),
	)

	keyConfirmYes = key.NewBinding(key.WithKeys("y", "enter"), key.WithHelp("y", "delete"))
	keyConfirmNo  = key.NewBinding(key.WithKeys("esc", "n"), key.WithHelp("esc/n", "cancel"))
)

func formSubmitHelp(action string) key.Binding {
	return key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "next/"+action))
}

func browsingKeyMap() table.KeyMap {
	km := table.DefaultKeyMap()
	return km
}

func choosingHelp() []key.Binding {
	return []key.Binding{keyUp, keyDown, keySelect, keyQuit}
}

func schemaHelp() []key.Binding {
	return []key.Binding{keyEscBack, keyQuit}
}

func formHelp(action string) []key.Binding {
	return []key.Binding{formSubmitHelp(action), keyFormFields, keyCancel}
}

func confirmHelp() []key.Binding {
	return []key.Binding{keyConfirmYes, keyConfirmNo}
}

func browsingHelp(km table.KeyMap) []key.Binding {
	return []key.Binding{
		keyInsert,
		keyUpdate,
		keyDelete,
		keySchema,
		keyBack,
		keyQuit,
		km.LineUp,
		km.LineDown,
		keyColsLeft,
		keyColsRight,
	}
}

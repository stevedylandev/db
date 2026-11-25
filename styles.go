package main

import (
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	selStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	baseStyle  = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("4"))

	tableHeaderStyle = table.DefaultStyles().Header.
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")).
				BorderBottom(true).
				Bold(false)
	tableSelectedStyle = table.DefaultStyles().Selected.
				Foreground(lipgloss.Color("8")).
				Background(lipgloss.Color("3")).
				Bold(false)
)

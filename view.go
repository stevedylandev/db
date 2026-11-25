package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
)

const (
	maxCellWidth = 24
	maxVisibleRows = 15
)

func (m model) View() tea.View {
	var b strings.Builder
	switch m.phase {
	case choosing:
		b.WriteString(m.viewChoosing())
	case viewing:
		b.WriteString(m.viewViewing())
	case inserting:
		b.WriteString(m.viewInserting())
	case browsing:
		b.WriteString(m.viewBrowsing())
	}
	if m.err != nil {
		b.WriteString("\nError: " + m.err.Error())
	}
	return tea.NewView(b.String())
}

func (m model) viewChoosing() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Select a table") + "\n\n")
	for i, t := range m.tables {
		cursor, line := "  ", t
		if i == m.cursor {
			cursor, line = "> ", selStyle.Render(t)
		}
		b.WriteString(cursor + line + "\n")
	}
	b.WriteString("\nup/down (k/j): move · enter: select · q: quit")
	return b.String()
}

func (m model) viewViewing() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Schema: "+m.table) + "\n\n")
	for _, c := range m.cols {
		pk := ""
		if c.pk == 1 {
			pk = " (pk)"
		}
		b.WriteString(fmt.Sprintf("  %-20s %s%s\n", c.name, strings.ToLower(c.ctype), pk))
	}
	if m.status != "" {
		b.WriteString("\n" + m.status + "\n")
	}
	b.WriteString("\nesc/enter: back · q: quit")
	return b.String()
}

func (m model) viewInserting() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Insert into "+m.table) + "\n\n")
	for i, c := range m.plan.cols {
		hint := strings.ToLower(c.ctype)
		if c.pk != 0 {
			hint += ", pk"
		}
		if c.dflt.Valid {
			hint += fmt.Sprintf(", default %s (empty = default)", c.dflt.String)
		}
		cursor, val := "  ", m.vals[i]
		if i == m.field {
			cursor, val = "> ", selStyle.Render(m.vals[i]+"▌")
		}
		b.WriteString(fmt.Sprintf("%s%-20s %-30s %s\n", cursor, c.name, hint, val))
	}
	if m.status != "" {
		b.WriteString("\n" + m.status + "\n")
	}
	b.WriteString("\ntype to edit · enter: next/insert · esc: cancel")
	return b.String()
}

func (m model) viewBrowsing() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Rows: "+m.table) + "\n\n")
	b.WriteString(baseStyle.Render(m.rowsTbl.View()) + "\n")
	b.WriteString("  " + m.rowsTbl.HelpView() + "\n")
	note := fmt.Sprintf("  %d rows", m.rowCount)
	if m.rowCount == readLimit {
		note += fmt.Sprintf(" (first %d)", readLimit)
	}
	b.WriteString(note + " · i: insert · s: schema · esc: back · q: quit")
	return b.String()
}

func newRowsTable(cols []col, data [][]string) table.Model {
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = len(c.name)
	}
	for _, r := range data {
		for i, v := range r {
			if i < len(widths) && len(v) > widths[i] {
				widths[i] = min(len(v), maxCellWidth)
			}
		}
	}

	columns := make([]table.Column, len(cols))
	total := 0
	for i, c := range cols {
		columns[i] = table.Column{Title: c.name, Width: widths[i]}
		total += widths[i]
	}
	rows := make([]table.Row, len(data))
	for i, r := range data {
		rows[i] = table.Row(r)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(min(max(len(rows), 1), maxVisibleRows)+2),
		table.WithWidth(total+2*len(cols)+2),
	)

	s := table.DefaultStyles()
	s.Header = tableHeaderStyle
	s.Selected = tableSelectedStyle
	t.SetStyles(s)
	return t
}

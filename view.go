package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

const (
	maxCellWidth   = 24
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
	case updating:
		b.WriteString(m.viewUpdating())
	case browsing:
		b.WriteString(m.viewBrowsing())
	case deleting:
		b.WriteString(m.viewDeleting())
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

func (m model) viewForm(title, action string) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(title) + "\n\n")
	for i, c := range m.plan.cols {
		parts := []string{strings.ToLower(c.ctype)}
		if c.pk != 0 {
			parts = append(parts, "pk")
		}
		if c.dflt.Valid {
			parts = append(parts, fmt.Sprintf("default %s (empty = default)", c.dflt.String))
		}
		b.WriteString("  " + formLabelStyle.Render(c.name))
		b.WriteString("  " + formHintStyle.Render("("+strings.Join(parts, " · ")+")") + "\n")
		b.WriteString(formInputStyle.Render(m.inputs[i].View()) + "\n\n")
	}
	if m.status != "" {
		b.WriteString("\n" + m.status + "\n")
	}
	b.WriteString("\ntype to edit · enter: next/" + action + " · up/down: fields · esc: cancel")
	return b.String()
}

func (m model) viewInserting() string {
	return m.viewForm("Insert into "+m.table, "insert")
}

func (m model) viewUpdating() string {
	return m.viewForm("Update row in "+m.table, "save")
}

func newInsertInputs(cols []col, values []string) []textinput.Model {
	inputs := make([]textinput.Model, len(cols))
	for i, c := range cols {
		ti := textinput.New()
		ti.Prompt = ""
		ti.SetWidth(40)
		if c.dflt.Valid && c.dflt.String != "" {
			ti.Placeholder = c.dflt.String
		}
		if i < len(values) {
			ti.SetValue(values[i])
		}
		inputs[i] = ti
	}
	return inputs
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
	b.WriteString(note + " · i: insert · u: update · d: delete · s: schema · esc: back · q: quit")
	return b.String()
}

func (m model) viewDeleting() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Rows: "+m.table) + "\n\n")
	b.WriteString(baseStyle.Render(m.rowsTbl.View()) + "\n")
	if m.status != "" {
		b.WriteString("\n" + m.status + "\n")
	}
	b.WriteString("\ndelete this row? y: delete · esc/n: cancel")
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

package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

const (
	maxCellWidth      = 24
	maxVisibleRows    = 15
	cellPadding       = 2
	roomAroundTable   = 4
	defaultTableWidth = 80
	minTableWidth     = 20
)

func (m model) View() tea.View {
	var b strings.Builder
	switch m.phase {
	case choosing:
		b.WriteString(m.viewChoosing())
	case schema:
		b.WriteString(m.viewSchema())
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
	b.WriteString("\n" + m.help.ShortHelpView(choosingHelp()))
	return b.String()
}

func (m model) viewSchema() string {
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
	b.WriteString("\n" + m.help.ShortHelpView(schemaHelp()))
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
		if len(c.enum) > 0 {
			parts = append(parts, "one of: "+strings.Join(c.enum, " / "))
		} else if c.dflt.Valid {
			parts = append(parts, fmt.Sprintf("default %s (empty = default)", c.dflt.String))
		}
		b.WriteString("  " + formLabelStyle.Render(c.name))
		b.WriteString("  " + formHintStyle.Render("("+strings.Join(parts, " · ")+")") + "\n")
		b.WriteString(formInputStyle.Render(m.inputs[i].View()) + "\n\n")
	}
	if m.status != "" {
		b.WriteString("\n" + m.status + "\n")
	}
	b.WriteString("\n" + m.help.ShortHelpView(formHelp(action)))
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
	title := "Rows: " + m.table
	note := fmt.Sprintf("%d rows", m.rowCount)
	if m.rowCount == readLimit {
		note += fmt.Sprintf(" (first %d)", readLimit)
	}
	b.WriteString(titleStyle.Render(title+" ("+note+")") + "\n\n")
	b.WriteString(baseStyle.Render(m.rowsTbl.View()) + "\n")
	b.WriteString("  " + m.help.ShortHelpView(browsingHelp(m.rowsTbl.KeyMap)) + "\n")
	return b.String()
}

func (m model) viewDeleting() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Rows: "+m.table) + "\n\n")
	b.WriteString(baseStyle.Render(m.rowsTbl.View()) + "\n")
	if m.status != "" {
		b.WriteString("\n" + m.status + "\n")
	}
	b.WriteString("\ndelete this row? " + m.help.ShortHelpView(confirmHelp()))
	return b.String()
}

func newRowsTable(cols []col, data [][]string, termW, termH, colOffset int) table.Model {
	columns := make([]table.Column, len(cols))
	for i, c := range cols {
		columns[i] = table.Column{Title: c.name}
	}
	rows := make([]table.Row, len(data))
	for i, r := range data {
		rows[i] = table.Row(r)
	}

	avail := tableAvailWidth(termW)
	columns = fitColumns(columns, rows, avail, colOffset)

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithKeyMap(browsingKeyMap()),
		table.WithFocused(true),
		table.WithHeight(visibleHeight(len(rows), termH)),
		table.WithWidth(avail),
	)

	s := table.DefaultStyles()
	s.Header = tableHeaderStyle
	s.Selected = tableSelectedStyle
	t.SetStyles(s)
	return t
}

func (m *model) refitRowsTable() {
	cols := m.rowsTbl.Columns()
	rows := m.rowsTbl.Rows()
	if len(cols) == 0 {
		return
	}
	avail := tableAvailWidth(m.width)
	m.rowsTbl.SetColumns(fitColumns(cols, rows, avail, m.colOffset))
	m.rowsTbl.SetWidth(avail)
	m.rowsTbl.SetHeight(visibleHeight(len(rows), m.height))
}

func (m model) columnWindow() (first, last int, ok bool) {
	cols := m.rowsTbl.Columns()
	first, last = -1, -1
	for i, c := range cols {
		if c.Width > 0 {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	return first, last, first >= 0
}

func tableAvailWidth(termW int) int {
	if termW <= 0 {
		return defaultTableWidth
	}
	return max(termW-roomAroundTable, minTableWidth)
}

func fitColumns(cols []table.Column, rows []table.Row, avail, first int) []table.Column {
	n := len(cols)
	if n == 0 {
		return cols
	}
	first = min(max(first, 0), n-1)
	natural := naturalWidths(cols, rows)
	total := 0
	for i := range cols {
		switch {
		case i < first:
			cols[i].Width = 0
		case i == first || total+natural[i]+cellPadding <= avail:
			cols[i].Width = natural[i]
			total += natural[i] + cellPadding
		default:
			cols[i].Width = 0
		}
	}
	if natural[first]+cellPadding > avail {
		cols[first].Width = max(avail-cellPadding, 1)
	}
	return cols
}

func naturalWidths(cols []table.Column, rows []table.Row) []int {
	n := len(cols)
	natural := make([]int, n)
	for i := range cols {
		natural[i] = max(len(cols[i].Title), 1)
	}
	for _, r := range rows {
		for i, v := range r {
			if i < n {
				natural[i] = max(natural[i], min(len(v), maxCellWidth))
			}
		}
	}
	return natural
}

func visibleHeight(n, termH int) int {
	visible := min(max(n, 1), maxVisibleRows)
	if termH > 0 {
		visible = min(visible, max(termH-8, 1))
	}
	return visible + 2
}

package main

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

func (m model) updateChoosing(key tea.KeyPressMsg) (model, tea.Cmd, bool) {
	switch key.String() {
	case "q":
		return m, tea.Quit, true
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.tables)-1 {
			m.cursor++
		}
	case "enter":
		m.table = m.tables[m.cursor]
		m.colOffset = 0
		cols, err := tableColumns(m.db, m.table)
		if err != nil {
			m.err = err
			return m, tea.Quit, true
		}
		m.cols = cols
		m.status = ""
		if err := m.loadRows(); err != nil {
			m.status = "read: " + err.Error()
			return m, nil, true
		}
		m.phase = browsing
	}
	return m, nil, false
}

func (m model) updateSchema(key tea.KeyPressMsg) (model, tea.Cmd, bool) {
	switch key.String() {
	case "q":
		return m, tea.Quit, true
	case "esc", "enter":
		m.phase = browsing
	}
	return m, nil, false
}

func (m model) updateForm(key tea.KeyPressMsg, submit func(model, []string) (model, tea.Cmd)) (model, tea.Cmd, bool) {
	switch s := key.String(); s {
	case "esc":
		m.status = ""
		m.phase = browsing
		return m, nil, true
	case "enter", "tab", "down":
		if m.field < len(m.inputs)-1 {
			return m, m.focusField(m.field + 1), true
		}
		if s != "enter" {
			return m, nil, true
		}
		vals := make([]string, len(m.inputs))
		for i := range vals {
			vals[i] = m.inputs[i].Value()
		}
		m2, cmd := submit(m, vals)
		return m2, cmd, true
	case "shift+tab", "up":
		if m.field > 0 {
			return m, m.focusField(m.field - 1), true
		}
		return m, nil, true
	}
	var cmd tea.Cmd
	m.inputs[m.field], cmd = m.inputs[m.field].Update(key)
	return m, cmd, true
}

func (m model) updateInserting(key tea.KeyPressMsg) (model, tea.Cmd, bool) {
	return m.updateForm(key, func(m model, vals []string) (model, tea.Cmd) {
		id, err := insertRow(m.db, m.plan, vals)
		if err != nil {
			m.status = "insert: " + err.Error()
			return m, nil
		}
		if err := m.loadRows(); err != nil {
			m.status = "read: " + err.Error()
			return m, nil
		}
		m.status = "1 row inserted"
		if m.plan.autoPk != "" {
			m.status += fmt.Sprintf(" (%s = %d)", m.plan.autoPk, id)
		}
		m.phase = browsing
		return m, nil
	})
}

func (m model) updateUpdating(key tea.KeyPressMsg) (model, tea.Cmd, bool) {
	return m.updateForm(key, func(m model, vals []string) (model, tea.Cmd) {
		n, err := updateRow(m.db, m.plan, vals, m.key)
		if err != nil {
			m.status = "update: " + err.Error()
			return m, nil
		}
		if err := m.loadRows(); err != nil {
			m.status = "read: " + err.Error()
			return m, nil
		}
		m.status = "row updated"
		if n != 1 {
			m.status = fmt.Sprintf("%d rows updated", n)
		}
		m.phase = browsing
		return m, nil
	})
}

func (m *model) focusField(i int) tea.Cmd {
	m.inputs[m.field].Blur()
	m.field = i
	return m.inputs[i].Focus()
}

func (m model) updateBrowsing(key tea.KeyPressMsg) (model, tea.Cmd, bool) {
	switch key.String() {
	case "q":
		return m, tea.Quit, true
	case "esc":
		m.phase = choosing
		return m, nil, true
	case "i":
		m.plan = planInsert(m.db, m.table, m.cols)
		if len(m.plan.cols) == 0 {
			m.status = "nothing to insert"
			return m, nil, true
		}
		m.inputs = newInsertInputs(m.plan.cols, nil)
		m.field = 0
		m.status = ""
		m.phase = inserting
		return m, m.inputs[0].Focus(), true
	case "u":
		row, ok := m.selectedRow()
		if !ok {
			m.status = "no row selected"
			return m, nil, true
		}
		rk, err := keyAt(m.cols, row, m.rowIds, m.rowsTbl.Cursor())
		if err != nil {
			m.status = "update: " + err.Error()
			return m, nil, true
		}
		m.plan = planInsert(m.db, m.table, m.cols)
		if len(m.plan.cols) == 0 {
			m.status = "nothing to update"
			return m, nil, true
		}
		m.key = rk
		m.inputs = newInsertInputs(m.plan.cols, formValues(m.cols, m.plan.cols, row))
		m.field = 0
		m.status = ""
		m.phase = updating
		return m, m.inputs[0].Focus(), true
	case "d":
		row, ok := m.selectedRow()
		if !ok {
			m.status = "no row selected"
			return m, nil, true
		}
		rk, err := keyAt(m.cols, row, m.rowIds, m.rowsTbl.Cursor())
		if err != nil {
			m.status = "delete: " + err.Error()
			return m, nil, true
		}
		m.key = rk
		m.status = ""
		m.phase = deleting
		return m, nil, true
	case "s":
		m.phase = schema
		return m, nil, true
	case "left", "h":
		if m.colOffset > 0 {
			m.colOffset--
			m.refitRowsTable()
		}
		return m, nil, true
	case "right", "l":
		// only scroll if there are hidden columns to the right
		if _, last, ok := m.columnWindow(); ok && last < len(m.cols)-1 {
			m.colOffset++
			m.refitRowsTable()
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m model) selectedRow() ([]string, bool) {
	if m.rowsTbl.Cursor() >= m.rowCount {
		return nil, false
	}
	row := m.rowsTbl.SelectedRow()
	if len(row) != len(m.cols) {
		return nil, false
	}
	return row, true
}

func formValues(cols, planCols []col, row []string) []string {
	vals := make([]string, len(planCols))
	for i, pc := range planCols {
		for j, c := range cols {
			if c.name == pc.name && j < len(row) {
				vals[i] = row[j]
				break
			}
		}
	}
	return vals
}

func (m model) updateDeleting(key tea.KeyPressMsg) (model, tea.Cmd, bool) {
	switch key.String() {
	case "y", "enter":
		n, err := deleteRow(m.db, m.table, m.key)
		if err != nil {
			m.status = "delete: " + err.Error()
			m.phase = browsing
			return m, nil, true
		}
		if err := m.loadRows(); err != nil {
			m.status = "read: " + err.Error()
			m.phase = browsing
			return m, nil, true
		}
		if n == 1 {
			m.status = "1 row deleted"
		} else {
			m.status = fmt.Sprintf("%d rows deleted", n)
		}
		m.phase = browsing
	case "esc", "n":
		m.status = ""
		m.phase = browsing
	}
	return m, nil, true
}

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

func (m model) updateViewing(key tea.KeyPressMsg) (model, tea.Cmd, bool) {
	switch key.String() {
	case "q":
		return m, tea.Quit, true
	case "esc", "enter":
		m.phase = browsing
	}
	return m, nil, false
}

func (m model) updateInserting(key tea.KeyPressMsg) (model, tea.Cmd, bool) {
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
		id, err := insertRow(m.db, m.plan, vals)
		if err != nil {
			m.status = "insert: " + err.Error()
			return m, nil, true
		}
		if err := m.loadRows(); err != nil {
			m.status = "read: " + err.Error()
			return m, nil, true
		}
		m.status = "1 row inserted"
		if m.plan.autoPk != "" {
			m.status += fmt.Sprintf(" (%s = %d)", m.plan.autoPk, id)
		}
		m.phase = browsing
		return m, nil, true
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
		m.inputs = newInsertInputs(m.plan.cols)
		m.field = 0
		m.status = ""
		m.phase = inserting
		return m, m.inputs[0].Focus(), true
	case "s":
		m.phase = viewing
		return m, nil, true
	}
	return m, nil, false
}

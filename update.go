package main

import (
	"fmt"
	"unicode/utf8"

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
	case "enter":
		if m.field < len(m.plan.cols)-1 {
			m.field++
			return m, nil, true
		}
		id, err := insertRow(m.db, m.plan, m.vals)
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
	case "backspace":
		if r := []rune(m.vals[m.field]); len(r) > 0 {
			m.vals[m.field] = string(r[:len(r)-1])
		}
	case "space":
		m.vals[m.field] += " "
	default:
		if utf8.RuneCountInString(s) == 1 {
			m.vals[m.field] += s
		}
	}
	return m, nil, false
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
		m.vals = make([]string, len(m.plan.cols))
		m.field = 0
		m.status = ""
		m.phase = inserting
	case "s":
		m.phase = viewing
		return m, nil, true
	}
	return m, nil, false
}

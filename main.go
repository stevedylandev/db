// db - a simple TUI for managing sqlite databases
//
// Todo:
// - Add Update and Delete methods
package main

import (
	"database/sql"
	"fmt"
	"os"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	_ "modernc.org/sqlite"
)

type phase int

const (
	choosing phase = iota
	viewing
	inserting
	browsing
)

type model struct {
	db     *sql.DB
	phase  phase
	tables []string
	cursor int
	table  string
	cols   []col
	err    error

	plan     insertPlan
	inputs   []textinput.Model
	field    int
	status   string
	rowsTbl  table.Model
	rowCount int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		if m.phase == inserting {
			var cmd tea.Cmd
			m.inputs[m.field], cmd = m.inputs[m.field].Update(msg)
			return m, cmd
		}
		return m, nil
	}
	if key.String() == "ctrl+c" {
		return m, tea.Quit
	}

	var (
		cmd     tea.Cmd
		handled bool
	)
	switch m.phase {
	case choosing:
		m, cmd, handled = m.updateChoosing(key)
	case viewing:
		m, cmd, handled = m.updateViewing(key)
	case inserting:
		m, cmd, handled = m.updateInserting(key)
	case browsing:
		m, cmd, handled = m.updateBrowsing(key)
	}
	if handled {
		return m, cmd
	}
	m.rowsTbl, cmd = m.rowsTbl.Update(key)
	return m, cmd
}

func (m *model) loadRows() error {
	rows, err := readRows(m.db, m.table, m.cols)
	if err != nil {
		return err
	}
	m.rowsTbl = newRowsTable(m.cols, rows)
	m.rowCount = len(rows)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: db <database file>")
		os.Exit(1)
	}
	db, err := sql.Open("sqlite", os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	tables, err := listTables(db)
	if err != nil {
		fmt.Println(err)
		return
	}

	p := tea.NewProgram(model{db: db, tables: tables})
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

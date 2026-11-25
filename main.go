// db - a simple TUI for managing sqlite databases
//
// Todo:
// - Migrate to Bubbletea
// - Add Update and Delete methods
// - Add Read methods?
package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	_ "modernc.org/sqlite"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	selStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
)

type phase int

const (
	choosing phase = iota
	viewing
	inserting
)

type model struct {
	db     *sql.DB
	phase  phase
	tables []string
	cursor int
	table  string
	cols   []col
	err    error

	plan   insertPlan 
	vals   []string
	field  int
	status string
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg: 
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.phase {
		case choosing:
			switch msg.String() {
			case "q":
				return m, tea.Quit
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
					return m, tea.Quit
				}
				m.cols = cols
				m.status = ""
				m.phase = viewing
			}

		case viewing:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "enter":
				m.phase = choosing 
			case "i":
				m.plan = planInsert(m.db, m.table, m.cols)
				if len(m.plan.cols) == 0 {
					m.status = "nothing to insert"
					return m, nil
				}
				m.vals = make([]string, len(m.plan.cols))
				m.field = 0
				m.status = ""
				m.phase = inserting
			}

		case inserting:
			switch s := msg.String(); s {
			case "esc":
				m.status = ""
				m.phase = viewing
			case "enter":
				if m.field < len(m.plan.cols)-1 {
					m.field++
					return m, nil
				}
				id, err := insertRow(m.db, m.plan, m.vals)
				if err != nil {
					m.status = "insert: " + err.Error()
					return m, nil 
				}
				m.status = "1 row inserted"
				if m.plan.autoPk != "" {
					m.status += fmt.Sprintf(" (%s = %d)", m.plan.autoPk, id)
				}
				m.phase = viewing
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
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	var b strings.Builder

	switch m.phase {
	case choosing:
		b.WriteString(titleStyle.Render("Select a table") + "\n\n")
		for i, t := range m.tables {
			cursor, line := "  ", t
			if i == m.cursor {
				cursor, line = "> ", selStyle.Render(t)
			}
			b.WriteString(cursor + line + "\n")
		}
		b.WriteString("\nup/down: move · enter: select · q: quit")

	case viewing:
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
		b.WriteString("\ni: insert · enter: back · q: quit")

	case inserting:
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
	}

	if m.err != nil {
		b.WriteString("\nError: " + m.err.Error())
	}
	return tea.NewView(b.String())
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

//
// db - a simple TUI for managing sqlite databases
//
// Todo: 
// - Migrate to Bubbletea
// - Add Update and Delete methods 
// - Add Read methods? 
//
package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

var in = bufio.NewScanner(os.Stdin)

type col struct {
	name    string
	ctype   string
	pk      int
	notNull int
	dflt    sql.NullString
	hidden  int
}

func ask(prompt string) string {
	fmt.Print(prompt)
	if !in.Scan() {
		fmt.Println()
		os.Exit(0)
	}
	return strings.TrimSpace(in.Text())
}

func ident(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func coerce(ctype, v string) any {
	t := strings.ToUpper(ctype)
	switch {
	case strings.Contains(t, "INT"):
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	case strings.Contains(t, "REAL"), strings.Contains(t, "FLOA"), strings.Contains(t, "DOUB"):
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return v
}

func listTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT name FROM sqlite_schema WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var next string
		if err := rows.Scan(&next); err != nil {
			return nil, err
		}
		out = append(out, next)
	}
	return out, rows.Err()
}

func tableColumns(db *sql.DB, table string) ([]col, error) {
	rows, err := db.Query("PRAGMA table_xinfo(" + ident(table) + ")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []col
	for rows.Next() {
		var cid int
		var c col
		if err := rows.Scan(&cid, &c.name, &c.ctype, &c.notNull, &c.dflt, &c.pk, &c.hidden); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func withoutRowid(db *sql.DB, table string) bool {
	var s sql.NullString
	if err := db.QueryRow(`SELECT sql FROM sqlite_schema WHERE type='table' AND name=?`, table).Scan(&s); err != nil || !s.Valid {
		return false
	}
	return strings.Contains(strings.ToUpper(s.String), "WITHOUT ROWID")
}

func singleIntPk(cols []col) (string, bool) {
	name := ""
	count := 0
	for _, c := range cols {
		if c.pk != 0 {
			count++
			name = c.name
		}
	}
	if count != 1 {
		return "", false
	}
	for _, c := range cols {
		if c.name == name && strings.EqualFold(strings.TrimSpace(c.ctype), "INTEGER") {
			return name, true
		}
	}
	return "", false
}

func doInsert(db *sql.DB, table string, cols []col) {
	autoPk, hasAutoPk := "", false
	if !withoutRowid(db, table) {
		autoPk, hasAutoPk = singleIntPk(cols)
	}

	var names, ph []string
	var args []any
	for _, c := range cols {
		if c.hidden != 0 { 
			fmt.Printf("  %s: skipped (generated)\n", c.name)
			continue
		}
		if hasAutoPk && c.name == autoPk {
			fmt.Printf("  %s: auto (INTEGER PRIMARY KEY)\n", c.name)
			continue
		}
		hint := strings.ToLower(c.ctype)
		if c.pk != 0 {
			hint += ", pk"
		}
		if c.dflt.Valid {
			hint += ", default " + c.dflt.String + " (enter = default)"
		}
		v := ask(fmt.Sprintf("  %s [%s]: ", c.name, hint))
		if v == "" && c.dflt.Valid {
			continue 
		}
		names = append(names, ident(c.name))
		ph = append(ph, "?")
		args = append(args, coerce(c.ctype, v))
	}
	if len(names) == 0 {
		fmt.Println("  nothing to insert")
		return
	}

	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		ident(table), strings.Join(names, ", "), strings.Join(ph, ", "))
	res, err := db.Exec(q, args...)
	if err != nil {
		fmt.Println("  err:", err)
		return
	}
	n, _ := res.RowsAffected()
	fmt.Printf("  %d row inserted\n", n)
	if hasAutoPk {
		if id, err := res.LastInsertId(); err == nil {
			fmt.Printf("  %s = %d\n", autoPk, id)
		}
	}
}

func main() {
	dbPath := os.Args[1]
	db, err := sql.Open("sqlite", dbPath)
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
	fmt.Println(tables)
	table := ask("Which table do you want to see? ")
	cols, err := tableColumns(db, table)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("\nSchema of " + table + ":")
	for _, c := range cols {
		pk := ""
		if c.pk == 1 {
			pk = " (pk)"
		}
		fmt.Printf("  %s %s%s\n", c.name, strings.ToLower(c.ctype), pk)
	}
	fmt.Println()
	doInsert(db, table, cols)
}

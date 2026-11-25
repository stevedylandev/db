package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

type col struct {
	name    string
	ctype   string
	pk      int
	notNull int
	dflt    sql.NullString
	hidden  int
}

const readLimit = 1000

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

func cell(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case []byte:
		return string(t)
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func readRows(db *sql.DB, table string, cols []col) ([][]string, error) {
	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = ident(c.name)
	}
	q := fmt.Sprintf("SELECT %s FROM %s LIMIT %d",
		strings.Join(names, ", "), ident(table), readLimit)

	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out [][]string
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		r := make([]string, len(cols))
		for i, v := range vals {
			r[i] = cell(v)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type insertPlan struct {
	table  string
	cols   []col
	autoPk string
}

func planInsert(db *sql.DB, table string, cols []col) insertPlan {
	p := insertPlan{table: table}
	if !withoutRowid(db, table) {
		p.autoPk, _ = singleIntPk(cols)
	}
	for _, c := range cols {
		if c.hidden != 0 {
			continue
		}
		if p.autoPk != "" && c.name == p.autoPk {
			continue
		}
		p.cols = append(p.cols, c)
	}
	return p
}

func insertRow(db *sql.DB, p insertPlan, values []string) (int64, error) {
	var names, ph []string
	var args []any
	for i, c := range p.cols {
		v := values[i]
		if v == "" && c.dflt.Valid {
			continue
		}
		names = append(names, ident(c.name))
		ph = append(ph, "?")
		args = append(args, coerce(c.ctype, v))
	}
	if len(names) == 0 {
		return 0, errors.New("nothing to insert")
	}

	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		ident(p.table), strings.Join(names, ", "), strings.Join(ph, ", "))
	res, err := db.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

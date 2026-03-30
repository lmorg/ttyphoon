package element_table

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/lmorg/ttyphoon/types"
	"golang.design/x/clipboard"
)

const _ROW_ID = "rowid"

func (el *ElementTable) sqlWhere() string {
	where := el.filter
	if where != "" {
		where = "WHERE " + where
	}

	return where
}

func (el *ElementTable) sqlString() string {
	orderBy := _ROW_ID
	var sql string
	if el.orderByIndex > 0 {
		orderBy = string(el.headings[el.orderByIndex-1])
		sql = sqlSelect[el.isNumber[el.orderByIndex-1]]
	} else {
		sql = sqlSelect[selectNumeric]
	}

	return fmt.Sprintf(sql, el.name, el.sqlWhere(), orderBy, orderByStr[el.orderDesc], el.size.Y-1, el.limitOffset)
}

func (el *ElementTable) runQuery() error {
	query := el.sqlString()
	dbRows, err := el.db.Query(query)
	if err != nil {
		return fmt.Errorf("cannot query table: %v\nSQL: %s", err, query)
	}

	var (
		table []string
		width = make([]int, len(el.headings))
		rows  [][]string
		l     = len(el.headings)
	)

	for dbRows.Next() {
		row := make([]string, l)
		slice := _strToAnyPtr(&row, l)

		err = dbRows.Scan(slice...)
		if err != nil {
			return err
		}

		for i := range row {
			if len([]rune(row[i])) > width[i] {
				width[i] = len([]rune(row[i]))
			}
		}

		rows = append(rows, row)
	}

	boundaries := make([]int32, len(el.headings))
	var boundaryPos int32
	// check if rows are smaller than headings
	// also lets do the boundaries for the table lines
	for i := range el.headings {
		if len(el.headings[i]) > width[i] {
			width[i] = len(el.headings[i])
		}
		boundaryPos += int32(width[i]) + 2
		boundaries[i] = boundaryPos
	}

	for _, row := range rows {
		var s string
		for i := range row {
			s += fmt.Sprintf(" %s%s ", row[i], strings.Repeat(" ", width[i]-len([]rune(row[i]))))
		}

		table = append(table, s)
	}

	var top string
	for i := range el.headings {
		top += fmt.Sprintf(" %s%s ", string(el.headings[i]), strings.Repeat(" ", width[i]-len(el.headings[i])))
	}

	if err = dbRows.Err(); err != nil {
		return fmt.Errorf("cannot retrieve rows: %v", err)
	}

	err = dbRows.Close()
	if err != nil {
		return err
	}

	el.table = make([][]rune, len(table))
	for i := range table {
		el.table[i] = []rune(table[i])
	}
	el.top = []rune(top)
	el.width = width
	el.boundaries = boundaries

	err = el.db.QueryRow(fmt.Sprintf(sqlCount, el.name, el.sqlWhere())).Scan(&el.lines)
	if err != nil {
		return fmt.Errorf("cannot get table count: %v", err)
	}

	return nil
}

func _strToAnyPtr(s *[]string, max int) []any {
	slice := make([]interface{}, max)
	for i := range slice {
		slice[i] = &(*s)[i]
	}

	return slice
}

func (el *ElementTable) exportHeadings() []string {
	headings := make([]string, len(el.headings))
	for i := range el.headings {
		headings[i] = string(el.headings[i])
	}

	return headings
}

func (el *ElementTable) exportRows() (string, [][]string, error) {
	query := el.sqlString()
	dbRows, err := el.db.Query(query)
	if err != nil {
		return query, nil, fmt.Errorf("cannot query table: %v", err)
	}
	defer func() {
		_ = dbRows.Close()
	}()

	rows := make([][]string, 0)
	l := len(el.headings)

	for dbRows.Next() {
		row := make([]string, l)
		slice := _strToAnyPtr(&row, l)

		err = dbRows.Scan(slice...)
		if err != nil {
			return query, nil, fmt.Errorf("cannot read table row: %v", err)
		}

		rows = append(rows, row)
	}

	if err = dbRows.Err(); err != nil {
		return query, nil, fmt.Errorf("cannot read table row: %v", err)
	}

	return query, rows, nil
}

func writeMarkdownTable(buf *bytes.Buffer, headings []string, rows [][]string) {
	if len(headings) == 0 {
		return
	}

	writeMarkdownRow := func(cols []string) {
		buf.WriteString("| ")
		for i := range cols {
			if i > 0 {
				buf.WriteString(" | ")
			}
			cell := strings.ReplaceAll(cols[i], "\\", "\\\\")
			cell = strings.ReplaceAll(cell, "|", "\\|")
			cell = strings.ReplaceAll(cell, "\n", "<br>")
			buf.WriteString(cell)
		}
		buf.WriteString(" |\n")
	}

	writeMarkdownRow(headings)

	buf.WriteString("| ")
	for i := range headings {
		if i > 0 {
			buf.WriteString(" | ")
		}
		buf.WriteString("---")
	}
	buf.WriteString(" |\n")

	for i := range rows {
		writeMarkdownRow(rows[i])
	}
}

func (el *ElementTable) ExportCsv() {
	var b []byte
	buf := bytes.NewBuffer(b)
	w := csv.NewWriter(buf)

	line := el.exportHeadings()

	err := w.Write(line)
	if err != nil {
		el.renderer.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("cannot read table row: %v", err))
		return
	}

	query, rows, err := el.exportRows()
	if err != nil {
		el.renderer.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("%v\nSQL: %s", err, query))
		return
	}

	for i := range rows {
		if err = w.Write(rows[i]); err != nil {
			el.renderer.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("cannot read table row: %v\nSQL: %s", err, query))
			return
		}
	}

	w.Flush()
	if err = w.Error(); err != nil {
		el.renderer.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("cannot read table row: %v\nSQL: %s", err, query))
		return
	}

	clipboard.Write(clipboard.FmtText, buf.Bytes())
}

func (el *ElementTable) ExportMarkdown() {
	query, rows, err := el.exportRows()
	if err != nil {
		el.renderer.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("%v\nSQL: %s", err, query))
		return
	}

	buf := bytes.NewBuffer(nil)
	writeMarkdownTable(buf, el.exportHeadings(), rows)
	clipboard.Write(clipboard.FmtText, buf.Bytes())
}

package element_table

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/lmorg/ttyphoon/types"
	"golang.design/x/clipboard"
)

// runQuery pulls the current view from the shared engine and lays it out for the
// terminal: column widths, boundaries and padded row strings.
func (el *ElementTable) runQuery() error {
	rows, err := el.core.Rows(int(el.size.Y-1), int(el.limitOffset))
	if err != nil {
		return err
	}

	var (
		table []string
		width = make([]int, len(el.headings))
	)

	for _, row := range rows {
		for i := range row {
			if len([]rune(row[i])) > width[i] {
				width[i] = len([]rune(row[i]))
			}
		}
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

	el.table = make([][]rune, len(table))
	for i := range table {
		el.table[i] = []rune(table[i])
	}
	el.top = []rune(top)
	el.width = width
	el.boundaries = boundaries

	count, err := el.core.Count()
	if err != nil {
		return err
	}
	el.lines = int32(count)

	return nil
}

func (el *ElementTable) exportHeadings() []string {
	headings := make([]string, len(el.headings))
	for i := range el.headings {
		headings[i] = string(el.headings[i])
	}

	return headings
}

func (el *ElementTable) exportRows() (string, [][]string, error) {
	rows, err := el.core.Rows(int(el.size.Y-1), int(el.limitOffset))
	if err != nil {
		return "", nil, err
	}

	return el.core.Name(), rows, nil
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

	clipboard.Write(context.Background(), clipboard.FmtText, buf.Bytes())
}

func (el *ElementTable) ExportMarkdown() {
	query, rows, err := el.exportRows()
	if err != nil {
		el.renderer.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("%v\nSQL: %s", err, query))
		return
	}

	buf := bytes.NewBuffer(nil)
	writeMarkdownTable(buf, el.exportHeadings(), rows)
	clipboard.Write(context.Background(), clipboard.FmtText, buf.Bytes())
}

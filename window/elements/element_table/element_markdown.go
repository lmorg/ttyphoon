package element_table

import "strings"

func fromMarkdown(el *ElementTable, params *parametersT) ([][]string, error) {
	var (
		table [][]string
		line  []string
		cell  []rune

		fstPipe = true
		escape  bool
		lineN   int
	)

	for _, r := range el.buf {
		switch r {

		case '\r', '\t':
			if escape {
				cell = append(cell, '\\')
				escape = false
			}
			continue

		case '\n':
			if escape {
				cell = append(cell, '\\')
				escape = false
			}
			fstPipe = true
			if len(line) == 0 {
				continue
			}
			lineN++
			if isMdSeparator(line) {
				if lineN == 1 {
					params.CreateHeadings = true
				}
				line = []string{}
				continue
			}
			table = append(table, line)
			line = []string{}

		case '\\':
			if escape {
				cell = append(cell, '\\')
				escape = false
			} else {
				escape = true
			}

		case '|':
			if escape {
				cell = append(cell, '|')
				escape = false
				continue
			}

			s := strings.TrimSpace(string(cell))
			if s == "" && fstPipe {
				fstPipe = false
				continue
			}
			line = append(line, s)
			cell = []rune{}

		default:
			if r == ' ' && fstPipe {
				continue
			}
			fstPipe = false
			cell = append(cell, r)
		}
	}

	if len(cell) > 0 {
		line = append(line, strings.TrimSpace(string(cell)))
	}
	if len(line) > 0 {
		table = append(table, line)
	}

	return table, nil
}

func isMdSeparator(line []string) bool {
	for _, r := range line[0] {
		if r == ' ' {
			continue
		}
		if r != '-' && r != ':' {
			return false
		}
	}
	return true
}

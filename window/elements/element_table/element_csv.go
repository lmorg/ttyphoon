package element_table

import (
	"bytes"
	"encoding/csv"
	"fmt"
)

func fromCsv(el *ElementTable) ([][]string, error) {
	buf := bytes.NewBufferString(string(el.buf))
	r := csv.NewReader(buf)
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1
	recs, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV: %v", err)
	}
	return recs, nil
}

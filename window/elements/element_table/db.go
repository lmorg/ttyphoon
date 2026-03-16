package element_table

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const driverName = "sqlite3"

const (
	sqlCreateTable     = `CREATE TABLE IF NOT EXISTS '%s' (%s);`
	sqlInsertRecord    = `INSERT INTO '%s' VALUES (%s);`
	sqlSelectByAlpha   = `SELECT * FROM '%s' %s ORDER BY lower("%s") %s LIMIT %d OFFSET %d;`
	sqlSelectByNumeric = `SELECT * FROM '%s' %s ORDER BY "%s" %s LIMIT %d OFFSET %d;`
	sqlCount           = `SELECT count(*) FROM '%s' %s;`
)

const (
	selectAlpha   = false
	selectNumeric = true
)

var sqlSelect = map[bool]string{
	selectAlpha:   sqlSelectByAlpha,
	selectNumeric: sqlSelectByNumeric,
}

var orderByStr = map[bool]string{
	false: "ASC",
	true:  "DESC",
}

func (el *ElementTable) createDb() error {
	var err error
	el.db, err = sql.Open(driverName, ":memory:")
	if err != nil {
		return fmt.Errorf("could not open database: %s", err.Error())
	}

	return nil
}

func (el *ElementTable) createTable(headings []string) error {
	var err error

	if len(headings) == 0 {
		return fmt.Errorf("cannot create table '%s': no titles supplied", el.name)
	}

	el.name = "csv"

	var sHeadings string
	for i := range headings {
		sHeadings += fmt.Sprintf(`"%s" NUMERIC,`, headings[i])
	}
	sHeadings = sHeadings[:len(sHeadings)-1]

	query := fmt.Sprintf(sqlCreateTable, el.name, sHeadings)
	_, err = el.db.Exec(query)
	if err != nil {
		return fmt.Errorf("could not create table '%s': %s\n%s", el.name, err.Error(), query)
	}

	el.dbTx, err = el.db.Begin()
	if err != nil {
		return fmt.Errorf("could not create transaction: %s", err.Error())
	}

	return nil
}

func (el *ElementTable) insertRecords(records []string) error {
	if len(records) == 0 {
		return fmt.Errorf("no records to insert into transaction on table %s", el.name)
	}

	values, err := _createValues(len(records))
	if err != nil {
		return fmt.Errorf("cannot insert records into transaction on table %s: %s", el.name, err.Error())
	}

	_, err = el.dbTx.Exec(fmt.Sprintf(sqlInsertRecord, el.name, values), _strToAny(records)...)
	if err != nil {
		return fmt.Errorf("cannot insert records into transaction on table %s: %s", el.name, err.Error())
	}

	return nil
}

func _strToAny(s []string) []any {
	a := make([]any, len(s))
	for i := 0; i < len(s); i++ {
		a[i] = s[i]
	}
	return a
}

func _createValues(length int) (string, error) {
	if length == 0 {
		return "", fmt.Errorf("no records to insert")
	}

	values := strings.Repeat("?,", length)
	values = values[:len(values)-1]

	return values, nil
}

package sbh

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/lmorg/mxtty/app"
	_ "github.com/mattn/go-sqlite3"
)

const driverName = "sqlite3"

const (
	sqlCreateTable  = `CREATE TABLE IF NOT EXISTS '%s' (%s);`
	sqlInsertRecord = `INSERT INTO '%s' VALUES (%s);`
)

func New(tileId string, errCallback func(error)) *ScrollbackHistory {
	sbh := &ScrollbackHistory{errCallback: errCallback}

	path := fmt.Sprintf("file:%s/%s-%d-scrollback-history-buf-%s-%d.db",
		os.TempDir(), app.Name, os.Getpid(), tileId, time.Now().Unix())
	db, err := sql.Open(driverName, path)
	if err != nil {
		errCallback(fmt.Errorf("cannot open database: %s", err.Error()))
		return sbh
	}

	_, err = db.Exec(`CREATE TABLE 'row' (
          id       INTEGER PRIMARY KEY,
          phrase   STRING  NOT NULL,  
          meta     INTEGER NOT NULL,
		  host     STRING  NOT NULL,
          pwd      STRING  NOT NULL,
          block_id INTEGER NOT NULL
       );`)
	if err != nil {
		errCallback(fmt.Errorf("cannot initialize table 'row': %s", err.Error()))
		return sbh
	}

	_, err = db.Exec(`CREATE TABLE 'block' (
          block_id INTEGER NOT NULL,
          exit_num INTEGER NOT NULL,
          query    STRING  NOT NULL,
          meta     INTEGER NOT NULL
       );`)
	if err != nil {
		errCallback(fmt.Errorf("cannot initialize table 'block': %s", err.Error()))
		return sbh
	}

	sbh.db = db
	return sbh
}

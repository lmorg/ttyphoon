package sbh

import (
	"database/sql"
	"fmt"

	"github.com/lmorg/ttyphoon/types"
)

type ScrollbackHistory struct {
	db          *sql.DB
	errCallback func(error)
}

func (sbh *ScrollbackHistory) Append(screen types.Screen) {
	if sbh.db == nil {
		return
	}

	tx, err := sbh.db.Begin()
	if err != nil {
		sbh.errCallback(fmt.Errorf("cannot start transaction: %s", err))
		sbh.db = nil
		return
	}

	for _, row := range screen {

		_, err = tx.Exec(`INSERT INTO 'row'
							(phrase, meta, host, pwd, block_id)
							VALUES (?, ?, ?, ?, ?);`,
			row.String(), row.RowMeta, row.Source.Host, row.Source.Pwd, row.Block.Id)
		if err != nil {
			sbh.errCallback(fmt.Errorf("cannot write into row_block transaction: %s", err))
			sbh.db = nil
			return
		}

		_, err = tx.Exec(`INSERT OR IGNORE INTO 'block'
							(block_id, exit_num, query, meta)
							VALUES (?, ?, ?, ?);`,
			row.Block.Id, row.Block.ExitNum, string(row.Block.Query), row.Block.Meta)
		if err != nil {
			sbh.errCallback(fmt.Errorf("cannot write into row_block transaction: %s", err))
			sbh.db = nil
			return
		}

	}

	err = tx.Commit()
	if err != nil {
		sbh.errCallback(fmt.Errorf("cannot commit transaction: %s", err))
		sbh.db = nil
		return
	}
}

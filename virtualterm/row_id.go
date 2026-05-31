package virtualterm

var rowId uint64

const UINT64_CAP = ^uint64(0)

func nextRowId() uint64 {
	if rowId == UINT64_CAP {
		rowId = 0
	} else {
		rowId++
	}

	return rowId
}

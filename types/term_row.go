package types

import (
	"errors"
	"strings"
)

type Row struct {
	Id      uint64
	Cells   []*Cell
	Hidden  Screen
	Source  *RowSource
	Block   *BlockMeta
	RowMeta RowMetaFlag
}

type RowMetaFlag uint16

const (
	META_ROW_NONE        RowMetaFlag = 0
	META_ROW_BEGIN_BLOCK RowMetaFlag = 1 << iota
	META_ROW_END_BLOCK
	META_ROW_FROM_LINE_OVERFLOW
	META_ROW_AUTO_HOTLINKED
)

func (f RowMetaFlag) Is(flag RowMetaFlag) bool { return f&flag != 0 }
func (f *RowMetaFlag) Set(flag RowMetaFlag)    { *f |= flag }
func (f *RowMetaFlag) Unset(flag RowMetaFlag)  { *f &^= flag }

type RowSource struct {
	Host string
	Pwd  string
}

type BlockMeta struct {
	Query   []rune // typically command line
	ExitNum int
	Meta    BlockMetaFlag
}

type BlockMetaFlag uint16

const (
	META_BLOCK_NONE BlockMetaFlag = 0
	META_BLOCK_OK   BlockMetaFlag = 1 << iota
	META_BLOCK_ERROR
	META_BLOCK_AI
)

func (f BlockMetaFlag) Is(flag BlockMetaFlag) bool { return f&flag != 0 }
func (f *BlockMetaFlag) Set(flag BlockMetaFlag)    { *f |= flag }
func (f *BlockMetaFlag) Unset(flag BlockMetaFlag)  { *f &^= flag }

func (r *Row) String() string {
	slice := make([]rune, len(r.Cells))

	for i := range r.Cells {
		slice[i] = r.Cells[i].Rune()
	}

	return string(slice)
}

/*
	SCREEN
*/

type Screen []*Row

func (screen *Screen) String() string {
	slice := make([]string, len(*screen))
	for i, row := range *screen {
		slice[i] = row.String()
	}

	return strings.Join(slice, "\n")
}

var (
	ERR_PHRASE_OVERFLOW_ROW = errors.New("overflow row")
	ERR_PHRASE_INVALID_ROW  = errors.New("index does not exist in slice")
)

func (screen *Screen) Phrase(row int) (string, error) {
	if row >= len(*screen) {
		return "", ERR_PHRASE_INVALID_ROW
	}
	if (*screen)[row].RowMeta.Is(META_ROW_FROM_LINE_OVERFLOW) {
		return "", ERR_PHRASE_OVERFLOW_ROW
	}

	slice := make([]rune, len((*screen)[row].Cells))

	for iCells := range (*screen)[row].Cells {
		slice[iCells] = (*screen)[row].Cells[iCells].Rune()
	}

	for iRow := row + 1; iRow < len(*screen); iRow++ {
		if !(*screen)[iRow].RowMeta.Is(META_ROW_FROM_LINE_OVERFLOW) {
			break
		}

		sliceRow := make([]rune, len((*screen)[iRow].Cells))
		for iCells := range (*screen)[iRow].Cells {
			sliceRow[iCells] = (*screen)[iRow].Cells[iCells].Rune()
		}

		slice = append(slice, sliceRow...)
	}

	return strings.TrimRight(string(slice), " "), nil
}

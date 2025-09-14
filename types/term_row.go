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
	META_ROW_AUTO_HYPERLINKED
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
	var (
		slice    []rune
		wideChar bool
		getChar  = func(c *Cell) rune {
			wideChar = c.Sgr != nil && c.Sgr.Bitwise.Is(SGR_WIDE_CHAR)
			return c.Rune()
		}
	)

	for _, c := range r.Cells {
		if wideChar {
			wideChar = false
			continue
		}
		slice = append(slice, getChar(c))
	}

	return strings.TrimLeft(string(slice), " ")
}

/*
	SCREEN
*/

type Screen []*Row

func (screen *Screen) String() string {
	var s string
	for _, row := range *screen {
		if !row.RowMeta.Is(META_ROW_FROM_LINE_OVERFLOW) {
			s += "\n"
		}
		s += row.String()
	}

	return s + "\n"
}

var (
	ErrPhraseOverflowRow = errors.New("overflow row")
	ErrPhraseInvalidRow  = errors.New("index does not exist in slice")
)

func (screen *Screen) Phrase(row int) (string, error) {
	if row >= len(*screen) {
		return "", ErrPhraseInvalidRow
	}
	if (*screen)[row].RowMeta.Is(META_ROW_FROM_LINE_OVERFLOW) {
		return "", ErrPhraseOverflowRow
	}

	var (
		slice    []rune
		wideChar bool
		getChar  = func(c *Cell) rune {
			wideChar = c.Sgr != nil && c.Sgr.Bitwise.Is(SGR_WIDE_CHAR)
			return c.Rune()
		}
	)

	for iCells := range (*screen)[row].Cells {
		if wideChar {
			wideChar = false
			continue
		}
		slice = append(slice, getChar((*screen)[row].Cells[iCells]))
	}

	for iRow := row + 1; iRow < len(*screen); iRow++ {
		if !(*screen)[iRow].RowMeta.Is(META_ROW_FROM_LINE_OVERFLOW) {
			break
		}

		wideChar = false
		for iCells := range (*screen)[iRow].Cells {
			if wideChar {
				wideChar = false
				continue
			}
			slice = append(slice, getChar((*screen)[iRow].Cells[iCells]))
		}
	}

	return strings.TrimRight(string(slice), " "), nil
}

func (screen *Screen) ContinuousRows(rowIndex int) []*Row {
	rows := make([]*Row, 0, len(*screen))

	for i := rowIndex; i >= 0; i-- {
		rows = append([]*Row{(*screen)[i]}, rows...)
		if !(*screen)[i].RowMeta.Is(META_ROW_FROM_LINE_OVERFLOW) {
			break
		}
	}

	for i := rowIndex + 1; i < len(*screen); i++ {
		if !(*screen)[i].RowMeta.Is(META_ROW_FROM_LINE_OVERFLOW) {
			break
		}
		rows = append(rows, (*screen)[i])
	}

	return rows
}

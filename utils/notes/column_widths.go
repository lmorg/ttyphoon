package notes

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/lmorg/ttyphoon/utils/cache"
)

func columnWidthCacheKey(filename, view string, headings []string, wrapped bool) string {
	normalized := make([]string, len(headings))
	for i := range headings {
		normalized[i] = strings.TrimSpace(headings[i])
	}

	hash := sha1.Sum([]byte(strings.Join(normalized, "\x1f")))
	tableID := hex.EncodeToString(hash[:])

	wrapBit := "0"
	if wrapped {
		wrapBit = "1"
	}

	return fmt.Sprintf("%s:%s:%s:%s", filename, tableID, strings.ToLower(strings.TrimSpace(view)), wrapBit)
}

func GetColumnWidths(filename, view string, headings []string, wrapped bool) []float64 {
	widths := make([]float64, 0)
	cache.Read(cache.NS_NOTESW_COLUMN_WIDTH, columnWidthCacheKey(filename, view, headings, wrapped), &widths)
	return widths
}

func SetColumnWidths(filename, view string, headings []string, wrapped bool, widths []float64) {
	if len(widths) == 0 {
		return
	}

	cache.Write(
		cache.NS_NOTESW_COLUMN_WIDTH,
		columnWidthCacheKey(filename, view, headings, wrapped),
		widths,
		cache.Days(30),
	)
}

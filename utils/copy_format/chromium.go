package copy_format

import (
	"encoding/binary"
	"unicode/utf16"
)

// ChromiumData encodes DataTransfer formats as a Chromium Pickle: a payload
// length, entry count, and length-prefixed UTF-16 strings aligned to four bytes.
func ChromiumData(formats Formats) []byte {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint32(data[4:], 3)
	appendString := func(value string) {
		units := utf16.Encode([]rune(value))
		data = binary.LittleEndian.AppendUint32(data, uint32(len(units)))
		for _, unit := range units {
			data = binary.LittleEndian.AppendUint16(data, unit)
		}
		for len(data)%4 != 0 {
			data = append(data, 0)
		}
	}
	for _, entry := range [][2]string{
		{"slack/html", formats.SlackHTML},
		{"text/html", formats.HTML},
		{"text/markdown", formats.Markdown},
	} {
		appendString(entry[0])
		appendString(entry[1])
	}
	binary.LittleEndian.PutUint32(data, uint32(len(data)-4))
	return data
}

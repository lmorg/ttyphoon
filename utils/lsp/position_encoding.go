package lsp

import "unicode/utf8"

// PositionEncoding names an LSP character-offset encoding.
type PositionEncoding string

const (
	PositionEncodingUTF8  PositionEncoding = "utf-8"
	PositionEncodingUTF16 PositionEncoding = "utf-16"
	PositionEncodingUTF32 PositionEncoding = "utf-32"
)

func normalizePositionEncoding(raw string) PositionEncoding {
	enc := PositionEncoding(raw)
	switch enc {
	case PositionEncodingUTF8, PositionEncodingUTF16, PositionEncodingUTF32:
		return enc
	default:
		return PositionEncodingUTF16
	}
}

func convertCharacterAtLine(content string, line, character int, from, to PositionEncoding) int {
	if character <= 0 || from == to {
		if character < 0 {
			return 0
		}
		return character
	}

	lineText, ok := lineTextAt(content, line)
	if !ok {
		return character
	}

	byteOffset := characterToByteOffset(lineText, character, from)
	return byteOffsetToCharacter(lineText, byteOffset, to)
}

func convertRangeAtURI(content string, r Range, from, to PositionEncoding) Range {
	if from == to {
		return r
	}

	r.Start.Character = convertCharacterAtLine(content, r.Start.Line, r.Start.Character, from, to)
	r.End.Character = convertCharacterAtLine(content, r.End.Line, r.End.Character, from, to)
	return r
}

func convertTextEdits(content string, edits []TextEdit, from, to PositionEncoding) []TextEdit {
	if from == to || len(edits) == 0 {
		return edits
	}

	out := make([]TextEdit, len(edits))
	for i := range edits {
		out[i] = edits[i]
		out[i].Range.Start.Character = convertCharacterAtLine(content, edits[i].Range.Start.Line, edits[i].Range.Start.Character, from, to)
		out[i].Range.End.Character = convertCharacterAtLine(content, edits[i].Range.End.Line, edits[i].Range.End.Character, from, to)
	}

	return out
}

func characterToByteOffset(lineText string, character int, enc PositionEncoding) int {
	if character <= 0 {
		return 0
	}

	switch enc {
	case PositionEncodingUTF8:
		if character >= len(lineText) {
			return len(lineText)
		}
		prev := 0
		for i := range lineText {
			if i == character {
				return i
			}
			if i > character {
				return prev
			}
			prev = i
		}
		if character > prev {
			return len(lineText)
		}
		return prev
	case PositionEncodingUTF32:
		col := 0
		for i := range lineText {
			if col >= character {
				return i
			}
			col++
		}
		return len(lineText)
	case PositionEncodingUTF16:
		col := 0
		for i, r := range lineText {
			if col >= character {
				return i
			}
			if r > 0xFFFF {
				col += 2
			} else {
				col++
			}
		}
		return len(lineText)
	default:
		return characterToByteOffset(lineText, character, PositionEncodingUTF16)
	}
}

func byteOffsetToCharacter(lineText string, byteOffset int, enc PositionEncoding) int {
	if byteOffset <= 0 {
		return 0
	}
	if byteOffset >= len(lineText) {
		switch enc {
		case PositionEncodingUTF8:
			return len(lineText)
		case PositionEncodingUTF32:
			return utf8.RuneCountInString(lineText)
		default:
			col := 0
			for _, r := range lineText {
				if r > 0xFFFF {
					col += 2
				} else {
					col++
				}
			}
			return col
		}
	}

	switch enc {
	case PositionEncodingUTF8:
		return byteOffset
	case PositionEncodingUTF32:
		col := 0
		for i := range lineText {
			if i >= byteOffset {
				return col
			}
			col++
		}
		return col
	case PositionEncodingUTF16:
		col := 0
		for i, r := range lineText {
			if i >= byteOffset {
				return col
			}
			if r > 0xFFFF {
				col += 2
			} else {
				col++
			}
		}
		return col
	default:
		return byteOffsetToCharacter(lineText, byteOffset, PositionEncodingUTF16)
	}
}

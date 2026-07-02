package spellcheck

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// SuggestionT represents a misspelled word with correction suggestions
type SuggestionT struct {
	MisspeltWord string   // The misspelled word
	WordStart    int      // Position where the word starts in the input
	WordLength   int      // Length of the misspelled word
	Suggestions  []string // List of suggested corrections
}

const MaxSuggestions = 5

// ExecAspell runs aspell -a with the given text and returns the raw output
func ExecAspell(text string) (string, error) {
	cmd := exec.Command("aspell", "-a")

	// Setup STDIN pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Setup STDOUT and STDERR capture
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start aspell: %w", err)
	}

	// Sanitize lines that start with aspell pipe-mode command characters
	// (#, -, !, %, ~, +, *, &, @, ^, $). We replace the first character with a
	// space — a 1-for-1 byte substitution — so every column offset in aspell's
	// output remains correct relative to the original text.
	sanitized := sanitizeForAspell(text)

	// Write text to stdin and close it
	if _, err := io.WriteString(stdin, sanitized); err != nil {
		stdin.Close()
		return "", fmt.Errorf("failed to write to stdin: %w", err)
	}
	stdin.Close()

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("aspell error: %s", stderr.String())
		}
		return "", fmt.Errorf("aspell failed: %w", err)
	}

	// Check for stderr even on success
	if stderr.Len() > 0 {
		return "", fmt.Errorf("aspell stderr: %s", stderr.String())
	}

	return stdout.String(), nil
}

// sanitizeForAspell replaces the first character of any line that begins with
// an aspell pipe-mode command character with a space. This prevents aspell
// from interpreting the line as a command. Because the replacement is exactly
// one byte for one byte, all column offsets reported by aspell remain correct
// relative to the original text.
func sanitizeForAspell(text string) string {
	// aspell treats these characters as commands when at the start of a line.
	const cmdChars = "*&@#!%~+-^$"
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if len(line) > 0 && strings.ContainsRune(cmdChars, rune(line[0])) {
			lines[i] = " " + line[1:]
		}
	}
	return strings.Join(lines, "\n")
}

// ParseAspellOutput parses the raw output from aspell -a and returns suggestions for misspelled words
func ParseAspellOutput(output string) ([]SuggestionT, error) {
	var suggestions []SuggestionT
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip version line (starts with @(#))
		if strings.HasPrefix(line, "@(#)") {
			continue
		}

		// Skip correct words (marked with *)
		if strings.HasPrefix(line, "*") {
			continue
		}

		// Parse misspelled word with suggestions (starts with &)
		if strings.HasPrefix(line, "& ") {
			suggestion, err := parseMisspelledLine(line)
			if err != nil {
				return nil, fmt.Errorf("failed to parse line '%s': %w", line, err)
			}
			suggestions = append(suggestions, suggestion)
			continue
		}

		// Parse misspelled word without suggestions (starts with #)
		if strings.HasPrefix(line, "# ") {
			suggestion, err := parseMisspelledLineNoSuggestions(line)
			if err != nil {
				return nil, fmt.Errorf("failed to parse line '%s': %w", line, err)
			}
			suggestions = append(suggestions, suggestion)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return suggestions, nil
}

// parseMisspelledLine parses a line like: & helo 23 0: hello, helot, help, ...
func parseMisspelledLine(line string) (SuggestionT, error) {
	// Remove the leading "& "
	line = strings.TrimPrefix(line, "& ")

	// Split at the colon to separate metadata from suggestions
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return SuggestionT{}, fmt.Errorf("invalid format: missing colon")
	}

	// Parse metadata: "helo 23 0"
	metadata := strings.Fields(parts[0])
	if len(metadata) != 3 {
		return SuggestionT{}, fmt.Errorf("invalid metadata format: expected 3 fields, got %d", len(metadata))
	}

	word := metadata[0]
	// count := metadata[1] // number of suggestions (not needed for our struct)
	offset, err := strconv.Atoi(metadata[2])
	if err != nil {
		return SuggestionT{}, fmt.Errorf("invalid offset: %w", err)
	}

	// Parse suggestions
	suggestionList := strings.Split(strings.TrimSpace(parts[1]), ", ")
	if len(suggestionList) > MaxSuggestions {
		suggestionList = suggestionList[:MaxSuggestions]
	}

	return SuggestionT{
		MisspeltWord: word,
		WordStart:    offset,
		WordLength:   len(word),
		Suggestions:  suggestionList,
	}, nil
}

// parseMisspelledLineNoSuggestions parses a line like: # word 0
func parseMisspelledLineNoSuggestions(line string) (SuggestionT, error) {
	// Remove the leading "# "
	line = strings.TrimPrefix(line, "# ")

	// Parse metadata: "word 0"
	metadata := strings.Fields(line)
	if len(metadata) != 2 {
		return SuggestionT{}, fmt.Errorf("invalid metadata format: expected 2 fields, got %d", len(metadata))
	}

	word := metadata[0]
	offset, err := strconv.Atoi(metadata[1])
	if err != nil {
		return SuggestionT{}, fmt.Errorf("invalid offset: %w", err)
	}

	return SuggestionT{
		MisspeltWord: word,
		WordStart:    offset,
		WordLength:   len(word),
		Suggestions:  []string{},
	}, nil
}

// ParseAspellMultilineOutput parses aspell -a output for multi-line input and
// returns suggestions with absolute character offsets into text.
//
// aspell pipe mode does NOT reliably emit one blank line per input line — it
// silently swallows lines starting with '#' (save-dict command), empty lines,
// and lines that contain no dictionary words.  Counting blank lines to infer
// the current input line is therefore unreliable.
//
// Instead we use the word text and aspell's per-line column offset to search
// forward through the input lines until we find the line where the word
// actually sits at that column.  Because aspell reports results in text order
// (top-to-bottom, left-to-right) we only ever scan forward, so the algorithm
// is O(lines + results).
func ParseAspellMultilineOutput(text, output string) ([]SuggestionT, error) {
	// Compute absolute start offset of each line.
	lines := strings.Split(text, "\n")
	lineStarts := make([]int, len(lines))
	off := 0
	for i, l := range lines {
		lineStarts[i] = off
		off += len(l) + 1 // +1 for '\n'
	}

	// Collect raw aspell results (WordStart is still the per-line column here).
	var raw []SuggestionT
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" ||
			strings.HasPrefix(line, "@(#)") ||
			strings.HasPrefix(line, "*") ||
			strings.HasPrefix(line, "-") {
			continue
		}
		var (
			s   SuggestionT
			err error
		)
		if strings.HasPrefix(line, "& ") {
			s, err = parseMisspelledLine(line)
		} else if strings.HasPrefix(line, "# ") {
			s, err = parseMisspelledLineNoSuggestions(line)
		} else {
			continue
		}
		if err != nil {
			return nil, err
		}
		raw = append(raw, s)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	// Match each raw result to the correct input line by searching forward.
	// prevLine/prevCol track the end of the last match so we never go backwards
	// (which correctly handles duplicate words at the same column on different
	// lines, or multiple misspellings on the same line).
	var results []SuggestionT
	prevLine, prevCol := 0, 0
	for _, s := range raw {
		col := s.WordStart // aspell per-line column offset
		word := s.MisspeltWord
		found := false
		for i := prevLine; i < len(lines); i++ {
			l := lines[i]
			// On the same line as the previous match, only accept columns
			// that come after the previous match's end position.
			if i == prevLine && col < prevCol {
				continue
			}
			if col+len(word) <= len(l) && l[col:col+len(word)] == word {
				s.WordStart = lineStarts[i] + col
				prevLine = i
				prevCol = col + len(word)
				found = true
				break
			}
		}
		if !found {
			continue
		}
		results = append(results, s)
	}
	return results, nil
}

// FilterExclusions removes suggestions for words in the exclusion list.
// The exclusion map should use lowercase keys for case-insensitive matching.
func FilterExclusions(suggestions []SuggestionT, exclusions map[string]bool) []SuggestionT {
	if len(exclusions) == 0 {
		return suggestions
	}

	filtered := make([]SuggestionT, 0, len(suggestions))
	for _, s := range suggestions {
		if !exclusions[strings.ToLower(s.MisspeltWord)] {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

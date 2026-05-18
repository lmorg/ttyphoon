package spelling

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

	// Write text to stdin and close it
	if _, err := io.WriteString(stdin, text); err != nil {
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

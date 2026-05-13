package spelling

import (
	"reflect"
	"testing"
)

func TestParseAspellOutput(t *testing.T) {
	// Example output from aspell -a
	input := `@(#) International Ispell Version 3.1.20 (but really Aspell 0.60.8.2)
& helo 23 0: hello, helot, help, halo, hell, heal, heel, held, helm, hero, he'll, Hal, Hale, hale, hole, Hall, Hill, Hull, hall, hill, holy, hula, hull
*
& wrold 64 8: world, wold, riled, roiled, rolled, told, weld, wild, would, Jerold, Rod, rod, ruled, old, reload, road, role, roll, rood, Wald, bold, cold, fold, gold, hold, railed, refold, resold, retold, sold, whorled, wield, Harold, reeled, wrote, wryly, warlord, Jerrold, rid, rode, roil, broiled, drooled, growled, prowled, trolled, Lord, RD, Rd, Roland, Ronald, lord, rd, clod, plod, relied, rowdy, relaid, LLD, paroled, rad, red, rel, rot`

	expected := []SuggestionT{
		{
			MisspeltWord: "helo",
			WordStart:    0,
			WordLength:   4,
			Suggestions: []string{
				"hello", "helot", "help", "halo", "hell", "heal", "heel", "held", "helm", "hero",
				"he'll", "Hal", "Hale", "hale", "hole", "Hall", "Hill", "Hull", "hall", "hill",
				"holy", "hula", "hull",
			},
		},
		{
			MisspeltWord: "wrold",
			WordStart:    8,
			WordLength:   5,
			Suggestions: []string{
				"world", "wold", "riled", "roiled", "rolled", "told", "weld", "wild", "would",
				"Jerold", "Rod", "rod", "ruled", "old", "reload", "road", "role", "roll", "rood",
				"Wald", "bold", "cold", "fold", "gold", "hold", "railed", "refold", "resold",
				"retold", "sold", "whorled", "wield", "Harold", "reeled", "wrote", "wryly",
				"warlord", "Jerrold", "rid", "rode", "roil", "broiled", "drooled", "growled",
				"prowled", "trolled", "Lord", "RD", "Rd", "Roland", "Ronald", "lord", "rd",
				"clod", "plod", "relied", "rowdy", "relaid", "LLD", "paroled", "rad", "red",
				"rel", "rot",
			},
		},
	}

	result, err := ParseAspellOutput(input)
	if err != nil {
		t.Fatalf("ParseAspellOutput returned error: %v", err)
	}

	if len(result) != len(expected) {
		t.Fatalf("Expected %d suggestions, got %d", len(expected), len(result))
	}

	for i, exp := range expected {
		res := result[i]

		if res.MisspeltWord != exp.MisspeltWord {
			t.Errorf("Suggestion %d: expected word '%s', got '%s'", i, exp.MisspeltWord, res.MisspeltWord)
		}

		if res.WordStart != exp.WordStart {
			t.Errorf("Suggestion %d: expected start %d, got %d", i, exp.WordStart, res.WordStart)
		}

		if res.WordLength != exp.WordLength {
			t.Errorf("Suggestion %d: expected length %d, got %d", i, exp.WordLength, res.WordLength)
		}

		if !reflect.DeepEqual(res.Suggestions, exp.Suggestions) {
			t.Errorf("Suggestion %d: suggestions mismatch\nExpected: %v\nGot: %v",
				i, exp.Suggestions, res.Suggestions)
		}
	}
}

func TestParseAspellOutput_CorrectWord(t *testing.T) {
	input := `@(#) International Ispell Version 3.1.20 (but really Aspell 0.60.8.2)
*
*
`

	result, err := ParseAspellOutput(input)
	if err != nil {
		t.Fatalf("ParseAspellOutput returned error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 suggestions for correct words, got %d", len(result))
	}
}

func TestParseAspellOutput_NoSuggestions(t *testing.T) {
	input := `@(#) International Ispell Version 3.1.20 (but really Aspell 0.60.8.2)
# xyzabc 5`

	result, err := ParseAspellOutput(input)
	if err != nil {
		t.Fatalf("ParseAspellOutput returned error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 suggestion, got %d", len(result))
	}

	exp := SuggestionT{
		MisspeltWord: "xyzabc",
		WordStart:    5,
		WordLength:   6,
		Suggestions:  []string{},
	}

	res := result[0]
	if res.MisspeltWord != exp.MisspeltWord {
		t.Errorf("Expected word '%s', got '%s'", exp.MisspeltWord, res.MisspeltWord)
	}

	if res.WordStart != exp.WordStart {
		t.Errorf("Expected start %d, got %d", exp.WordStart, res.WordStart)
	}

	if res.WordLength != exp.WordLength {
		t.Errorf("Expected length %d, got %d", exp.WordLength, res.WordLength)
	}

	if len(res.Suggestions) != 0 {
		t.Errorf("Expected no suggestions, got %v", res.Suggestions)
	}
}

func TestParseMisspelledLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected SuggestionT
		wantErr  bool
	}{
		{
			name:  "valid line with multiple suggestions",
			input: "& helo 23 0: hello, helot, help",
			expected: SuggestionT{
				MisspeltWord: "helo",
				WordStart:    0,
				WordLength:   4,
				Suggestions:  []string{"hello", "helot", "help"},
			},
			wantErr: false,
		},
		{
			name:  "valid line with single suggestion",
			input: "& teh 1 5: the",
			expected: SuggestionT{
				MisspeltWord: "teh",
				WordStart:    5,
				WordLength:   3,
				Suggestions:  []string{"the"},
			},
			wantErr: false,
		},
		{
			name:    "invalid line missing colon",
			input:   "& helo 23 0 hello",
			wantErr: true,
		},
		{
			name:    "invalid line missing offset",
			input:   "& helo 23: hello",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseMisspelledLine(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Result mismatch\nExpected: %+v\nGot: %+v", tt.expected, result)
			}
		})
	}
}

func TestParseMisspelledLineNoSuggestions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected SuggestionT
		wantErr  bool
	}{
		{
			name:  "valid line",
			input: "# xyzabc 0",
			expected: SuggestionT{
				MisspeltWord: "xyzabc",
				WordStart:    0,
				WordLength:   6,
				Suggestions:  []string{},
			},
			wantErr: false,
		},
		{
			name:  "valid line with offset",
			input: "# qwerty 10",
			expected: SuggestionT{
				MisspeltWord: "qwerty",
				WordStart:    10,
				WordLength:   6,
				Suggestions:  []string{},
			},
			wantErr: false,
		},
		{
			name:    "invalid line missing offset",
			input:   "# word",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseMisspelledLineNoSuggestions(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Result mismatch\nExpected: %+v\nGot: %+v", tt.expected, result)
			}
		})
	}
}

// Note: ExecAspell tests are not included as they require aspell to be installed.
// To test ExecAspell manually, run:
//
//   func TestExecAspell_Integration(t *testing.T) {
//       if testing.Short() {
//           t.Skip("Skipping integration test")
//       }
//
//       output, err := ExecAspell("helo wrold\n")
//       if err != nil {
//           t.Fatalf("ExecAspell failed: %v", err)
//       }
//
//       if output == "" {
//           t.Error("Expected non-empty output")
//       }
//
//       suggestions, err := ParseAspellOutput(output)
//       if err != nil {
//           t.Fatalf("ParseAspellOutput failed: %v", err)
//       }
//
//       if len(suggestions) == 0 {
//           t.Error("Expected at least one suggestion")
//       }
//   }

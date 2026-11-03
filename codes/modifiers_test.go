package codes_test

import (
	"testing"

	"github.com/lmorg/ttyphoon/codes"
	"github.com/lmorg/ttyphoon/types"
)

func TestGetAnsiEscSeqWithModifer(t *testing.T) {
	b := codes.GetAnsiEscSeq(types.KeysNormal, codes.AnsiF5, codes.MOD_SHIFT)
	if string(b) != string(codes.Csi)+"15;2~" {
		t.Errorf("Incorrect string '%s'", string(b))
	}
}

func TestEndKeyNameToSequence(t *testing.T) {
	keyCode, modifier := codes.KeyNameToCode("End")
	if keyCode != codes.AnsiEnd || modifier != 0 {
		t.Fatalf("End lookup = (%d, %d), want (%d, 0)", keyCode, modifier, codes.AnsiEnd)
	}
	for _, test := range []struct {
		name string
		mode types.KeyboardMode
		want string
	}{
		{"normal", types.KeysNormal, "\x1b[F"},
		{"application", types.KeysApplication, "\x1bOF"},
		{"vt220", types.KeysVT220, "\x1b[4~"},
		{"tmux", types.KeysTmuxClient, "\x00End "},
	} {
		t.Run(test.name, func(t *testing.T) {
			sequence := codes.GetAnsiEscSeq(test.mode, keyCode, modifier)
			if string(sequence) != test.want {
				t.Fatalf("End sequence = %q, want %q", sequence, test.want)
			}
			if test.mode == types.KeysTmuxClient && codes.TmuxKeySanitiser(sequence) != "End" {
				t.Fatalf("tmux End token = %q, want %q", codes.TmuxKeySanitiser(sequence), "End")
			}
		})
	}
}

func TestEndKeyNameModifiers(t *testing.T) {
	for name, want := range map[codes.KeyName]codes.Modifier{
		"C-End": codes.MOD_CTRL,
		"S-End": codes.MOD_SHIFT,
		"A-End": codes.MOD_ALT,
		"M-End": codes.MOD_META,
	} {
		keyCode, modifier := codes.KeyNameToCode(name)
		if keyCode != codes.AnsiEnd || modifier != want {
			t.Errorf("%s lookup = (%d, %d), want (%d, %d)", name, keyCode, modifier, codes.AnsiEnd, want)
		}
	}
}

func TestGetAnsiEscSeq_EndKeyNormalMode(t *testing.T) {
	b := codes.GetAnsiEscSeq(types.KeysNormal, codes.AnsiEnd, 0)
	// End key in normal cursor mode must be CSI F, not CSI E (which is CNL).
	const want = "\x1b[F"
	if string(b) != want {
		t.Errorf("End key normal mode: got %q, want %q", string(b), want)
	}
}

func TestGetAnsiEscSeq_EndKeyApplicationMode(t *testing.T) {
	b := codes.GetAnsiEscSeq(types.KeysApplication, codes.AnsiEnd, 0)
	// End key in application cursor mode must be SS3 F, not SS3 E.
	const want = "\x1bOF"
	if string(b) != want {
		t.Errorf("End key application mode: got %q, want %q", string(b), want)
	}
}

func TestGetAnsiEscSeq_HomeKeyNormalMode(t *testing.T) {
	b := codes.GetAnsiEscSeq(types.KeysNormal, codes.AnsiHome, 0)
	const want = "\x1b[H"
	if string(b) != want {
		t.Errorf("Home key normal mode: got %q, want %q", string(b), want)
	}
}

func TestGetAnsiEscSeq_HomeKeyApplicationMode(t *testing.T) {
	b := codes.GetAnsiEscSeq(types.KeysApplication, codes.AnsiHome, 0)
	const want = "\x1bOH"
	if string(b) != want {
		t.Errorf("Home key application mode: got %q, want %q", string(b), want)
	}
}

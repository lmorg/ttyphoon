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

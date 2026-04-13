package codes_test

import (
	"testing"

	"github.com/lmorg/ttyphoon/codes"
	"github.com/lmorg/ttyphoon/types"
)

func TestGetMouseEscSeqXtermPress(t *testing.T) {
	seq := codes.GetMouseEscSeq(types.KeysNormal, codes.MouseEvent{
		Type:   codes.MouseEventPress,
		Button: types.MOUSE_BUTTON_LEFT,
		X:      9,
		Y:      4,
	})

	if string(seq) != "\x1b[<0;10;5M" {
		t.Fatalf("unexpected sequence: %q", string(seq))
	}
}

func TestGetMouseEscSeqXtermRelease(t *testing.T) {
	seq := codes.GetMouseEscSeq(types.KeysNormal, codes.MouseEvent{
		Type:   codes.MouseEventRelease,
		Button: types.MOUSE_BUTTON_LEFT,
		X:      2,
		Y:      3,
	})

	if string(seq) != "\x1b[<3;3;4m" {
		t.Fatalf("unexpected sequence: %q", string(seq))
	}
}

func TestGetMouseEscSeqTmux(t *testing.T) {
	seq := codes.GetMouseEscSeq(types.KeysTmuxClient, codes.MouseEvent{
		Type:   codes.MouseEventWheelUp,
		Button: types.MOUSE_BUTTON_LEFT,
	})

	if string(seq) != "\x00WheelUpPane " {
		t.Fatalf("unexpected sequence: %q", string(seq))
	}
}

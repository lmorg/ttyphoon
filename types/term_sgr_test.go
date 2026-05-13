package types

import "testing"

func TestUnderlineBits(t *testing.T) {
	tests := []struct {
		name string
		flag SgrFlag
		want UnderlineStyle
	}{
		{
			name: "none",
			flag: 0,
			want: 0,
		},
		{
			name: "single bit only",
			flag: SGR_UNDERLINE,
			want: 1,
		},
		{
			name: "second bit only",
			flag: _SGR_UNDERLINE_2nd_bit,
			want: 2,
		},
		{
			name: "third bit only",
			flag: _SGR_UNDERLINE_3rd_bit,
			want: 4,
		},
		{
			name: "all underline bits set",
			flag: SGR_UNDERLINE | _SGR_UNDERLINE_2nd_bit | _SGR_UNDERLINE_3rd_bit,
			want: 7,
		},
		{
			name: "underline bits plus unrelated flags",
			flag: SGR_BOLD | SGR_UNDERLINE | _SGR_UNDERLINE_3rd_bit,
			want: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.flag.GetUnderlineStyle()
			if got != tt.want {
				t.Fatalf("UnderlineBits() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSetUnderlineStyle(t *testing.T) {
	tests := []struct {
		name  string
		style UnderlineStyle
	}{
		{name: "none", style: UNDERLINE_NONE},
		{name: "single", style: UNDERLINE_SINGLE},
		{name: "double", style: UNDERLINE_DOUBLE},
		{name: "curly", style: UNDERLINE_CURLY},
		{name: "dotted", style: UNDERLINE_DOTTED},
		{name: "dashed", style: UNDERLINE_DASHED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := SGR_BOLD | SGR_ITALIC
			flag.SetUnderlineStyle(tt.style)

			if got := flag.GetUnderlineStyle(); got != tt.style {
				t.Fatalf("UnderlineStyle() = %d, want %d", got, tt.style)
			}

			if !flag.Is(SGR_BOLD) || !flag.Is(SGR_ITALIC) {
				t.Fatalf("SetUnderlineStyle() unexpectedly modified unrelated bits")
			}
		})
	}
}

func TestSetUnderlineStyle_MasksToThreeBits(t *testing.T) {
	flag := SgrFlag(0)
	flag.SetUnderlineStyle(UnderlineStyle(9))

	if got := flag.GetUnderlineStyle(); got != 1 {
		t.Fatalf("UnderlineStyle() = %d, want %d", got, 1)
	}
}

package runebuf

import (
	"io"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRuneLength(t *testing.T) {
	tests := []struct {
		Rune   string
		Length int
	}{
		{
			Rune:   "0",
			Length: 1,
		},
		{
			Rune:   "世",
			Length: 3,
		},
		{
			Rune:   "界",
			Length: 3,
		},
		{
			Rune:   "🤗",
			Length: 4,
		},
	}

	for i, test := range tests {
		act := runeLength(test.Rune[0])
		if act != test.Length {
			t.Errorf("Unexpected length of rune i test %d", i)
			t.Logf("  Rune:     %s", test.Rune)
			t.Logf("  Expected: %d", test.Length)
			t.Logf("  Actual:   %d", act)
		}
	}
}

func TestBufSizeTracksUnreadUTF8Bytes(t *testing.T) {
	buf := New()
	defer buf.Close()
	input := "A界🙂"
	buf.Write([]byte(input))
	if got := buf.BufSize(); got != len(input) {
		t.Fatalf("initial BufSize() = %d, want %d", got, len(input))
	}
	for _, expected := range input {
		got, err := buf.Read()
		if err != nil || got != expected {
			t.Fatalf("Read() = %q, %v; want %q", got, err, expected)
		}
		wantPending := buf.BufSize()
		if wantPending < 0 {
			t.Fatalf("BufSize() became negative: %d", wantPending)
		}
	}
	if got := buf.BufSize(); got != 0 {
		t.Fatalf("BufSize() after draining = %d, want 0", got)
	}
}

func TestReadWaitsForSplitUTF8Sequence(t *testing.T) {
	buf := New()
	defer buf.Close()
	sequence := []byte("界")
	buf.Write(sequence[:2])
	if got := buf.BufSize(); got != 2 {
		t.Fatalf("partial sequence BufSize() = %d, want 2", got)
	}

	read := make(chan struct {
		r   rune
		err error
	}, 1)
	go func() {
		r, err := buf.Read()
		read <- struct {
			r   rune
			err error
		}{r, err}
	}()
	buf.Write(sequence[2:])
	select {
	case result := <-read:
		if result.err != nil || result.r != '界' {
			t.Fatalf("Read() = %q, %v; want 界", result.r, result.err)
		}
	case <-time.After(time.Second):
		t.Fatal("Read() did not wake after the UTF-8 sequence completed")
	}
	if got := buf.BufSize(); got != 0 {
		t.Fatalf("BufSize() after reading split sequence = %d, want 0", got)
	}
}

func TestInvalidBytesDoNotKeepBufSizePending(t *testing.T) {
	buf := New()
	defer buf.Close()
	buf.Write([]byte{0xff, 'x'})
	r, err := buf.Read()
	if err != nil || r != 'x' {
		t.Fatalf("Read() = %q, %v; want x after skipping invalid byte", r, err)
	}
	if got := buf.BufSize(); got != 0 {
		t.Fatalf("BufSize() after invalid byte was skipped = %d, want 0", got)
	}
}

func TestCloseWakesBlockedReader(t *testing.T) {
	buf := New()
	result := make(chan error, 1)
	go func() {
		_, err := buf.Read()
		result <- err
	}()
	buf.Close()
	select {
	case err := <-result:
		if err != io.EOF {
			t.Fatalf("Read() error = %v, want EOF", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close() did not wake blocked reader")
	}
}

func TestBufSizeCountsOnlyUnreadRunes(t *testing.T) {
	buf := New()
	defer buf.Close()
	buf.Write([]byte(strings.Repeat("x", 3)))
	for range 3 {
		if _, err := buf.Read(); err != nil {
			t.Fatal(err)
		}
	}
	if got := buf.BufSize(); got != 0 {
		t.Fatalf("BufSize() after all runes were consumed = %d, want 0", got)
	}
}

func TestConcurrentWritesReadsAndBufSize(t *testing.T) {
	buf := New()
	defer buf.Close()
	const chunks, chunkSize = 64, 128
	total := chunks * chunkSize
	payload := []byte(strings.Repeat("x", chunkSize))
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for i := 0; i < chunks; i++ {
			buf.Write(payload)
		}
	}()

	pollStop := make(chan struct{})
	pollErr := make(chan int, 1)
	go func() {
		for {
			select {
			case <-pollStop:
				return
			default:
				if pending := buf.BufSize(); pending < 0 {
					pollErr <- pending
					return
				}
				runtime.Gosched()
			}
		}
	}()

	for i := 0; i < total; i++ {
		r, err := buf.Read()
		if err != nil || r != 'x' {
			t.Fatalf("Read() at %d = %q, %v; want x", i, r, err)
		}
	}
	<-writerDone
	close(pollStop)
	select {
	case pending := <-pollErr:
		t.Fatalf("BufSize() became negative: %d", pending)
	default:
	}
	if got := buf.BufSize(); got != 0 {
		t.Fatalf("BufSize() after concurrent drain = %d, want 0", got)
	}
}

package runebuf

import (
	"bytes"
	"io"
	"log"

	"github.com/lmorg/mxtty/codes"
)

type Buf struct {
	bytes chan []byte
	r     chan rune
	utf8  []byte
	l     int
}

func New() *Buf {
	buf := &Buf{
		bytes: make(chan []byte),
		r:     make(chan rune),
	}

	go buf.loop()

	return buf
}

func (buf *Buf) loop() {
	for {
		b, ok := <-buf.bytes
		if !ok {
			return
		}

		for i := 0; i < len(b); i++ {
			if buf.l == 0 {
				buf.l = runeLength(b[i])
				if buf.l == 0 {
					log.Printf("ERROR: skipping invalid byte: %d", b[i])
					//panic("excellent")
					continue
				}
				buf.utf8 = make([]byte, buf.l)
			}

			buf.utf8[len(buf.utf8)-buf.l] = b[i]

			if buf.l == 1 {
				buf.r <- bytes.Runes(buf.utf8)[0]
			}
			buf.l--
		}
	}
}

func runeLength(b byte) int {
	switch {
	case b&128 == 0:
		return 1
	case b&32 == 0:
		return 2
	case b&16 == 0:
		return 3
	case b&8 == 0:
		return 4
	default:
		return 0
	}
}

func (buf *Buf) Write(b []byte) {
	buf.bytes <- b
}

func (buf *Buf) Read() (rune, error) {
	r, ok := <-buf.r
	if ok {
		return r, nil
	}

	return codes.AsciiEOF, io.EOF
}

func (buf *Buf) Close() {
	close(buf.bytes)
	close(buf.r) // TODO: should really allow the channel to flush first
}

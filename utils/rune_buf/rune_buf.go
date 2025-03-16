package runebuf

import (
	"bytes"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lmorg/mxtty/codes"
)

type Buf struct {
	bytes  []byte
	bm     sync.Mutex
	runes  []rune
	rm     sync.Mutex
	utf8   []byte
	l      int
	rPtr   int
	closed atomic.Bool
}

func New() *Buf {
	buf := &Buf{}

	go buf.loop()

	return buf
}

func (buf *Buf) loop() {
	for {
		if buf.closed.Load() {
			return
		}

		buf.bm.Lock()
		if len(buf.bytes) == 0 {
			buf.bm.Unlock()
			time.Sleep(15 * time.Millisecond)
			continue
		}

		b := make([]byte, len(buf.bytes))
		copy(b, buf.bytes)
		buf.bytes = []byte{}

		buf.bm.Unlock()

		for i := 0; i < len(b); i++ {
			if buf.l == 0 {
				buf.l = runeLength(b[i])
				if buf.l == 0 {
					log.Printf("ERROR: skipping invalid byte: %d", b[i])
					continue
				}
				buf.utf8 = make([]byte, buf.l)
			}

			buf.utf8[len(buf.utf8)-buf.l] = b[i]

			if buf.l == 1 {
				buf.rm.Lock()
				buf.runes = append(buf.runes, bytes.Runes(buf.utf8)[0])
				buf.rm.Unlock()
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
	buf.bm.Lock()
	buf.bytes = append(buf.bytes, b...)
	buf.bm.Unlock()
}

func (buf *Buf) Read() (rune, error) {
	for {
		if buf.closed.Load() {
			return codes.AsciiEOF, io.EOF
		}

		buf.rm.Lock()
		if len(buf.runes) == 0 {
			buf.rm.Unlock()
			time.Sleep(15 * time.Millisecond)
			continue
		}

		if buf.rPtr != 0 && buf.rPtr >= len(buf.runes) {
			buf.runes = []rune{}
			buf.rPtr = 0
			buf.rm.Unlock()
			time.Sleep(15 * time.Millisecond)
			continue
		}

		r := buf.runes[buf.rPtr]
		buf.inc()
		buf.rm.Unlock()
		return r, nil
	}
}

func (buf *Buf) inc() {
	buf.rPtr++
}

func (buf *Buf) BufSize() int {
	buf.rm.Lock()
	size := len(buf.runes)
	buf.rm.Unlock()
	return size
}

func (buf *Buf) Close() {
	//close(buf.bytes)
	//close(buf.r) // TODO: should really allow the channel to flush first
	buf.closed.Store(true)
}

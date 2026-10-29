package runebuf

import (
	"io"
	"log"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"github.com/lmorg/ttyphoon/codes"
)

const decodedRuneBatchSize = 256

type Buf struct {
	bytes       []byte
	bm          sync.Mutex
	runes       []rune
	runeBytes   []uint8
	rm          sync.Mutex
	runeReady   *sync.Cond
	rPtr        int
	writeReady  chan struct{}
	closeCh     chan struct{}
	closeOnce   sync.Once
	pending     atomic.Int64
	partialUTF8 [utf8.UTFMax]byte
	partialLen  int
	closed      atomic.Bool
}

func New() *Buf {
	buf := &Buf{
		writeReady: make(chan struct{}, 1),
		closeCh:    make(chan struct{}),
	}
	buf.runeReady = sync.NewCond(&buf.rm)

	go buf.loop()

	return buf
}

func (buf *Buf) loop() {
	for {
		if buf.closed.Load() {
			return
		}

		select {
		case <-buf.writeReady:
		case <-buf.closeCh:
			return
		}

		for {
			buf.bm.Lock()
			if buf.closed.Load() {
				buf.bm.Unlock()
				return
			}
			if len(buf.bytes) == 0 {
				buf.bm.Unlock()
				break
			}

			// Transfer ownership rather than copying the whole backlog while
			// holding the producer lock.
			chunk := buf.bytes
			buf.bytes = nil
			buf.bm.Unlock()
			buf.decode(chunk)
		}
	}
}

func (buf *Buf) decode(chunk []byte) {
	data := chunk
	if buf.partialLen > 0 {
		joined := make([]byte, 0, buf.partialLen+len(chunk))
		joined = append(joined, buf.partialUTF8[:buf.partialLen]...)
		joined = append(joined, chunk...)
		data = joined
		buf.partialLen = 0
	}
	var decodedRunes [decodedRuneBatchSize]rune
	var decodedBytes [decodedRuneBatchSize]uint8
	decodedCount := 0
	flush := func() {
		if decodedCount == 0 {
			return
		}
		buf.rm.Lock()
		buf.runes = append(buf.runes, decodedRunes[:decodedCount]...)
		buf.runeBytes = append(buf.runeBytes, decodedBytes[:decodedCount]...)
		buf.runeReady.Signal()
		buf.rm.Unlock()
		decodedCount = 0
	}

	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 && !utf8.FullRune(data) {
			buf.partialLen = copy(buf.partialUTF8[:], data)
			flush()
			return
		}
		if r == utf8.RuneError && size == 1 {
			log.Printf("ERROR: skipping invalid byte: %d", data[0])
			buf.pending.Add(-1)
			data = data[size:]
			continue
		}

		decodedRunes[decodedCount] = r
		decodedBytes[decodedCount] = uint8(size)
		decodedCount++
		if decodedCount == decodedRuneBatchSize {
			flush()
		}
		data = data[size:]
	}
	flush()
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
	if len(b) == 0 {
		return
	}
	buf.bm.Lock()
	if buf.closed.Load() {
		buf.bm.Unlock()
		return
	}
	buf.pending.Add(int64(len(b)))
	buf.bytes = append(buf.bytes, b...)
	buf.bm.Unlock()
	select {
	case buf.writeReady <- struct{}{}:
	default:
	}
}

func (buf *Buf) Read() (rune, error) {
	if buf.closed.Load() {
		return codes.AsciiEOF, io.EOF
	}

	buf.rm.Lock()
	for buf.rPtr >= len(buf.runes) && !buf.closed.Load() {
		buf.runeReady.Wait()
	}
	if buf.closed.Load() {
		buf.rm.Unlock()
		return codes.AsciiEOF, io.EOF
	}

	r := buf.runes[buf.rPtr]
	buf.pending.Add(-int64(buf.runeBytes[buf.rPtr]))
	buf.rPtr++
	if buf.rPtr == len(buf.runes) {
		buf.runes = buf.runes[:0]
		buf.runeBytes = buf.runeBytes[:0]
		buf.rPtr = 0
	} else {
		buf.runeReady.Signal()
	}
	buf.rm.Unlock()
	return r, nil
}

func (buf *Buf) BufSize() int {
	return int(buf.pending.Load())
}

func (buf *Buf) Close() {
	buf.closeOnce.Do(func() {
		buf.bm.Lock()
		buf.closed.Store(true)
		buf.bm.Unlock()
		close(buf.closeCh)
		buf.rm.Lock()
		buf.runeReady.Broadcast()
		buf.rm.Unlock()
	})
}

package agent

import (
	"fmt"
	"sync"
	"time"

	"github.com/lmorg/ttyphoon/ai/agent/sessiondb"
)

type aiStreamBlockWriter struct {
	workspace string
	sessionID int64
	runID     uint64
	emit      func(sessiondb.AIStreamBlock)

	mu          sync.Mutex
	nextOrdinal int64
	nextID      uint64
	text        *aiStreamBlockHandle
	thinking    *aiStreamBlockHandle
	notice      *aiStreamBlockHandle
}

type aiStreamBlockHandle struct {
	writer  *aiStreamBlockWriter
	block   sessiondb.AIStreamBlock
	mu      sync.Mutex
	closed  bool
	pending string
	timer   *time.Timer
}

const streamBlockFlushInterval = 250 * time.Millisecond

func newAIStreamBlockWriter(workspace string, emit func(sessiondb.AIStreamBlock)) *aiStreamBlockWriter {
	identity := sessiondb.GetActiveStreamIdentity(workspace)
	if identity.SessionID <= 0 || identity.RunID == 0 {
		return nil
	}
	return &aiStreamBlockWriter{
		workspace: workspace,
		sessionID: identity.SessionID,
		runID:     identity.RunID,
		emit:      emit,
		nextID:    1,
	}
}

func (w *aiStreamBlockWriter) Open(kind sessiondb.StreamBlockKind, parentID string) *aiStreamBlockHandle {
	return w.OpenWithLabel(kind, parentID, "")
}

func (w *aiStreamBlockWriter) OpenWithLabel(kind sessiondb.StreamBlockKind, parentID, label string) *aiStreamBlockHandle {
	if w == nil {
		return nil
	}
	if kind != sessiondb.StreamBlockThinking {
		w.CloseCurrentThinking()
	}
	w.mu.Lock()
	ordinal := w.nextOrdinal
	w.nextOrdinal++
	blockID := fmt.Sprintf("%d-%d", w.runID, w.nextID)
	w.nextID++
	w.mu.Unlock()

	block := sessiondb.AIStreamBlock{
		SessionID: w.sessionID,
		RunID:     w.runID,
		Workspace: w.workspace,
		BlockID:   blockID,
		ParentID:  parentID,
		Kind:      kind,
		Label:     label,
		Ordinal:   ordinal,
		Status:    "open",
	}
	if err := sessiondb.CreateStreamBlock(w.workspace, block, streamBlockNow()); err != nil {
		return nil
	}
	w.emitBlock(block)
	return &aiStreamBlockHandle{writer: w, block: block}
}

func (w *aiStreamBlockWriter) AppendText(text string) {
	if w == nil || text == "" {
		return
	}
	w.mu.Lock()
	block := w.text
	w.mu.Unlock()
	if block == nil || block.isClosed() {
		block = w.Open(sessiondb.StreamBlockText, "")
		w.mu.Lock()
		if w.text == nil || w.text.isClosed() {
			w.text = block
		} else {
			block.Close()
			block = w.text
		}
		w.mu.Unlock()
	}
	if block != nil {
		block.Append(text)
	}
}

func (w *aiStreamBlockWriter) AppendThinking(text string) {
	if w == nil || text == "" {
		return
	}
	w.mu.Lock()
	block := w.thinking
	w.mu.Unlock()
	if block == nil || block.isClosed() {
		block = w.Open(sessiondb.StreamBlockThinking, "")
		w.mu.Lock()
		if w.thinking == nil || w.thinking.isClosed() {
			w.thinking = block
		} else {
			block.Close()
			block = w.thinking
		}
		w.mu.Unlock()
	}
	if block != nil {
		block.Append(text)
	}
}

func (w *aiStreamBlockWriter) AppendNotice(text string) {
	if w == nil || text == "" {
		return
	}
	w.mu.Lock()
	block := w.notice
	w.mu.Unlock()
	if block == nil || block.isClosed() {
		block = w.Open(sessiondb.StreamBlockNotice, "")
		w.mu.Lock()
		if w.notice == nil || w.notice.isClosed() {
			w.notice = block
		} else {
			block.Close()
			block = w.notice
		}
		w.mu.Unlock()
	}
	if block != nil {
		block.Append(text)
	}
}

func (w *aiStreamBlockWriter) AppendStandalone(kind sessiondb.StreamBlockKind, text, parentID string) string {
	return w.AppendStandaloneWithLabel(kind, text, parentID, "")
}

func (w *aiStreamBlockWriter) AppendStandaloneWithLabel(kind sessiondb.StreamBlockKind, text, parentID, label string) string {
	if w == nil || text == "" {
		return ""
	}
	block := w.OpenWithLabel(kind, parentID, label)
	if block == nil {
		return ""
	}
	block.Append(text)
	block.Close()
	return block.block.BlockID
}

func (w *aiStreamBlockWriter) CloseCurrentThinking() {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.thinking != nil {
		w.thinking.Close()
	}
}

func (w *aiStreamBlockWriter) CloseAll() {
	if w == nil {
		return
	}
	w.mu.Lock()
	blocks := []*aiStreamBlockHandle{w.text, w.thinking, w.notice}
	w.mu.Unlock()
	for _, block := range blocks {
		if block != nil {
			block.Close()
		}
	}
}

func (h *aiStreamBlockHandle) Append(delta string) {
	if h == nil || h.writer == nil || delta == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.block.Content += delta
	h.pending += delta
	event := h.block
	event.Content = ""
	event.Delta = delta
	h.writer.emitBlock(event)
	if h.timer == nil {
		h.timer = time.AfterFunc(streamBlockFlushInterval, h.flush)
	}
}

func (h *aiStreamBlockHandle) Close() {
	if h == nil || h.writer == nil {
		return
	}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	if h.timer != nil {
		h.timer.Stop()
		h.timer = nil
	}
	pending := h.pending
	h.pending = ""
	h.mu.Unlock()
	if err := sessiondb.AppendStreamBlock(h.writer.workspace, h.block.SessionID, h.block.RunID, h.block.BlockID, pending, "closed", streamBlockNow()); err == nil {
		event := h.block
		event.Content = ""
		event.Status = "closed"
		h.writer.emitBlock(event)
	}
}

func (h *aiStreamBlockHandle) flush() {
	if h == nil || h.writer == nil {
		return
	}
	h.mu.Lock()
	if h.closed || h.pending == "" {
		h.timer = nil
		h.mu.Unlock()
		return
	}
	pending := h.pending
	h.pending = ""
	h.timer = nil
	err := sessiondb.AppendStreamBlock(h.writer.workspace, h.block.SessionID, h.block.RunID, h.block.BlockID, pending, "", streamBlockNow())
	if err != nil {
		h.pending = pending + h.pending
	}
	h.mu.Unlock()
	if err != nil {
		h.mu.Lock()
		if !h.closed && h.timer == nil {
			h.timer = time.AfterFunc(streamBlockFlushInterval, h.flush)
		}
		h.mu.Unlock()
	}
}

func (h *aiStreamBlockHandle) isClosed() bool {
	if h == nil {
		return true
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.closed
}

func streamBlockNow() string {
	return time.Now().Format("2006-01-02 15:04:05.000000000Z07:00")
}

func (w *aiStreamBlockWriter) emitBlock(block sessiondb.AIStreamBlock) {
	if w != nil && w.emit != nil {
		w.emit(block)
	}
}

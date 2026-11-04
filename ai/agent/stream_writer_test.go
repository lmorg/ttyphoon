package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/lmorg/ttyphoon/ai/agent/sessiondb"
	"github.com/lmorg/ttyphoon/app"
)

func TestAIStreamBlockWriter_ConcurrentParentedBlocks(t *testing.T) {
	workspace := "stream-writer-concurrency-test"
	state, err := sessiondb.CreateSession(workspace, "", 24)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	path := filepath.Join(home, "Documents", app.DirName, "session."+workspace+".db")
	t.Cleanup(func() { _ = os.Remove(path) })

	if err := sessiondb.CreateStreamPrompt(workspace, state.ActiveSessionID, 0, 99, "request", "start"); err != nil {
		t.Fatalf("CreateStreamPrompt: %v", err)
	}

	var eventsMu sync.Mutex
	var events []sessiondb.AIStreamBlock
	writer := &aiStreamBlockWriter{
		workspace: workspace,
		sessionID: state.ActiveSessionID,
		runID:     99,
		emit: func(block sessiondb.AIStreamBlock) {
			eventsMu.Lock()
			events = append(events, block)
			eventsMu.Unlock()
		},
	}

	parent := writer.Open(sessiondb.StreamBlockSubagent, "")
	if parent == nil {
		t.Fatal("Open parent returned nil")
	}
	child := writer.Open(sessiondb.StreamBlockText, parent.block.BlockID)
	if child == nil {
		t.Fatal("Open child returned nil")
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			parent.Append("parent ")
		}()
		go func() {
			defer wg.Done()
			child.Append("child ")
		}()
	}
	wg.Wait()
	if parent.block.Content != "" || child.block.Content != "" {
		t.Fatalf("writer retained accumulated content: parent=%d child=%d", len(parent.block.Content), len(child.block.Content))
	}
	eventsMu.Lock()
	for _, event := range events {
		if event.Delta != "" {
			eventsMu.Unlock()
			t.Fatalf("Append emitted an unbatched live delta: %+v", event)
		}
	}
	eventsMu.Unlock()
	parent.Close()
	child.Close()

	eventsMu.Lock()
	deltas := map[string]string{}
	for _, event := range events {
		if event.Delta != "" {
			deltas[event.BlockID] += event.Delta
		}
	}
	eventsMu.Unlock()
	if deltas[parent.block.BlockID] != strings.Repeat("parent ", 20) {
		t.Fatalf("parent emitted delta = %q, want 20 parent chunks", deltas[parent.block.BlockID])
	}
	if deltas[child.block.BlockID] != strings.Repeat("child ", 20) {
		t.Fatalf("child emitted delta = %q, want 20 child chunks", deltas[child.block.BlockID])
	}

	if err := sessiondb.FinalizeStreamPrompt(workspace, state.ActiveSessionID, 99, 42, "finish", sessiondb.PromptStatusSuccess); err != nil {
		t.Fatalf("FinalizeStreamPrompt: %v", err)
	}

	blocks, err := sessiondb.ListStreamBlockMeta(workspace, state.ActiveSessionID, 42)
	if err != nil {
		t.Fatalf("ListStreamBlockMeta: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("blocks = %d, want 2", len(blocks))
	}
	if blocks[1].ParentID != blocks[0].BlockID {
		t.Fatalf("child parent = %q, want %q", blocks[1].ParentID, blocks[0].BlockID)
	}

	parentContent, err := sessiondb.GetStreamBlockContent(workspace, state.ActiveSessionID, 42, blocks[0].BlockID)
	if err != nil {
		t.Fatalf("GetStreamBlockContent parent: %v", err)
	}
	childContent, err := sessiondb.GetStreamBlockContent(workspace, state.ActiveSessionID, 42, blocks[1].BlockID)
	if err != nil {
		t.Fatalf("GetStreamBlockContent child: %v", err)
	}
	if parentContent != strings.Repeat("parent ", 20) {
		t.Fatalf("parent content = %q, want 20 parent chunks", parentContent)
	}
	if childContent != strings.Repeat("child ", 20) {
		t.Fatalf("child content = %q, want 20 child chunks", childContent)
	}
}

func TestAIStreamBlockWriter_ScopedThinkingSurvivesSiblingAgentBlocks(t *testing.T) {
	workspace := "stream-writer-scoped-thinking-test"
	state, err := sessiondb.CreateSession(workspace, "", 24)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	path := filepath.Join(home, "Documents", app.DirName, "session."+workspace+".db")
	t.Cleanup(func() { _ = os.Remove(path) })

	var eventsMu sync.Mutex
	writer := &aiStreamBlockWriter{
		workspace: workspace,
		sessionID: state.ActiveSessionID,
		runID:     101,
		emit: func(sessiondb.AIStreamBlock) {
			eventsMu.Lock()
			eventsMu.Unlock()
		},
	}

	alpha := writer.Open(sessiondb.StreamBlockSubagent, "")
	bravo := writer.Open(sessiondb.StreamBlockSubagent, "")
	if alpha == nil || bravo == nil {
		t.Fatal("Open subagent blocks returned nil")
	}
	alphaScope := alpha.block.BlockID
	bravoScope := bravo.block.BlockID

	writer.AppendThinking("alpha reasons ", alphaScope)
	alphaThinking := writer.rolling[rollingBlockKey{kind: sessiondb.StreamBlockThinking, scope: alphaScope}]
	if alphaThinking == nil {
		t.Fatal("alpha has no thinking block")
	}

	// A sibling agent's tool call must not truncate alpha's reasoning.
	writer.Open(sessiondb.StreamBlockToolCall, bravoScope).Close()
	writer.AppendThinking("and keeps reasoning", alphaScope)

	stillAlpha := writer.rolling[rollingBlockKey{kind: sessiondb.StreamBlockThinking, scope: alphaScope}]
	if stillAlpha != alphaThinking {
		t.Fatal("sibling agent's tool call closed another agent's thinking block")
	}
	if alphaThinking.isClosed() {
		t.Fatal("alpha thinking block was closed by a sibling agent")
	}

	// Alpha's own tool call still ends alpha's reasoning.
	writer.Open(sessiondb.StreamBlockToolCall, alphaScope).Close()
	if !alphaThinking.isClosed() {
		t.Fatal("alpha's own tool call did not close its thinking block")
	}

	writer.AppendThinking("bravo reasons", bravoScope)
	bravoThinking := writer.rolling[rollingBlockKey{kind: sessiondb.StreamBlockThinking, scope: bravoScope}]
	if bravoThinking == nil || bravoThinking == alphaThinking {
		t.Fatal("bravo did not get its own thinking block")
	}
	if bravoThinking.block.ParentID != bravoScope {
		t.Fatalf("bravo thinking parent = %q, want %q", bravoThinking.block.ParentID, bravoScope)
	}

	writer.CloseAll()
	alpha.Close()
	bravo.Close()

	if err := sessiondb.FinalizeStreamPrompt(workspace, state.ActiveSessionID, 101, 43, "finish", sessiondb.PromptStatusSuccess); err != nil {
		t.Fatalf("FinalizeStreamPrompt: %v", err)
	}
	alphaContent, err := sessiondb.GetStreamBlockContent(workspace, state.ActiveSessionID, 43, alphaThinking.block.BlockID)
	if err != nil {
		t.Fatalf("GetStreamBlockContent alpha thinking: %v", err)
	}
	if alphaContent != "alpha reasons and keeps reasoning" {
		t.Fatalf("alpha thinking content = %q, want the uninterrupted sentence", alphaContent)
	}
}

func TestAIStreamBlockWriter_ConcurrentScopedThinkingKeepsEveryDelta(t *testing.T) {
	workspace := "stream-writer-concurrent-thinking-test"
	state, err := sessiondb.CreateSession(workspace, "", 24)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	path := filepath.Join(home, "Documents", app.DirName, "session."+workspace+".db")
	t.Cleanup(func() { _ = os.Remove(path) })

	writer := &aiStreamBlockWriter{
		workspace: workspace,
		sessionID: state.ActiveSessionID,
		runID:     102,
		emit:      func(sessiondb.AIStreamBlock) {},
	}

	scopes := []string{"", "agent-a", "agent-b", "agent-c"}
	marker := func(scope string) string {
		if scope == "" {
			return "root"
		}
		return scope
	}
	var wg sync.WaitGroup
	for _, scope := range scopes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				writer.AppendThinking(marker(scope)+"|", scope)
				if i%10 == 0 {
					// Interleave a tool call from a different agent.
					writer.Open(sessiondb.StreamBlockToolCall, "agent-noise").Close()
				}
			}
		}()
	}
	wg.Wait()
	writer.CloseAll()

	if err := sessiondb.FinalizeStreamPrompt(workspace, state.ActiveSessionID, 102, 44, "finish", sessiondb.PromptStatusSuccess); err != nil {
		t.Fatalf("FinalizeStreamPrompt: %v", err)
	}
	blocks, err := sessiondb.ListStreamBlockMeta(workspace, state.ActiveSessionID, 44)
	if err != nil {
		t.Fatalf("ListStreamBlockMeta: %v", err)
	}

	totals := map[string]int{}
	for _, meta := range blocks {
		if meta.Kind != sessiondb.StreamBlockThinking {
			continue
		}
		content, err := sessiondb.GetStreamBlockContent(workspace, state.ActiveSessionID, 44, meta.BlockID)
		if err != nil {
			t.Fatalf("GetStreamBlockContent: %v", err)
		}
		for _, chunk := range strings.Split(content, "|") {
			if chunk == "" {
				continue
			}
			if chunk != marker(meta.ParentID) {
				t.Fatalf("thinking block for scope %q contains another agent's text %q", meta.ParentID, chunk)
			}
			totals[chunk]++
		}
	}
	for _, scope := range scopes {
		if totals[marker(scope)] != 50 {
			t.Fatalf("scope %q recorded %d reasoning chunks, want 50", scope, totals[marker(scope)])
		}
	}
}

func TestAIStreamBlockWriter_SplitsThinkingAroundToolBlock(t *testing.T) {
	workspace := "stream-writer-thinking-boundary-test"
	state, err := sessiondb.CreateSession(workspace, "", 24)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	path := filepath.Join(home, "Documents", app.DirName, "session."+workspace+".db")
	t.Cleanup(func() { _ = os.Remove(path) })

	var events []sessiondb.AIStreamBlock
	writer := &aiStreamBlockWriter{
		workspace: workspace,
		sessionID: state.ActiveSessionID,
		runID:     100,
		emit: func(block sessiondb.AIStreamBlock) {
			events = append(events, block)
		},
	}

	writer.AppendThinking("before tool", "")
	firstThinking := writer.rolling[rollingBlockKey{kind: sessiondb.StreamBlockThinking}]
	if firstThinking == nil {
		t.Fatal("AppendThinking did not open a thinking block")
	}
	firstThinkingID := firstThinking.block.BlockID
	tool := writer.Open(sessiondb.StreamBlockToolCall, "")
	if tool == nil {
		t.Fatal("Open tool returned nil")
	}
	tool.Append("input")
	tool.Close()
	writer.AppendThinking("after tool", "")

	secondThinking := writer.rolling[rollingBlockKey{kind: sessiondb.StreamBlockThinking}]
	if secondThinking == nil || secondThinking.block.BlockID == tool.block.BlockID || secondThinking.block.BlockID == firstThinkingID {
		t.Fatal("thinking block was not reopened after the tool block")
	}
	closedBeforeTool := false
	for _, event := range events {
		if event.BlockID == firstThinkingID && event.Status == "closed" {
			closedBeforeTool = true
			break
		}
	}
	if !closedBeforeTool {
		t.Fatalf("events = %+v, want thinking close before tool block", events)
	}
	for _, event := range events {
		if event.Delta != "" && event.Content != "" {
			t.Fatalf("live delta event copied full content: %+v", event)
		}
	}
}

func TestStreamBlockWriterFromContextReusesRunWriter(t *testing.T) {
	writer := &aiStreamBlockWriter{runID: 123}
	outerContext := withAIStreamCallback(context.Background(), &aiStreamEmitter{blocks: writer})

	firstWindowWriter := streamBlockWriterFromContext(outerContext)
	secondWindowEmitter := newChildAIStreamEmitter(outerContext, nil)
	if firstWindowWriter != writer || secondWindowEmitter.blocks != writer {
		t.Fatal("bounded windows did not reuse the run-scoped stream block writer")
	}
}

func TestEinoAgentToolKeepsToolCallBlockOpenUntilReturn(t *testing.T) {
	workspace := "stream-writer-tool-call-lifecycle-test"
	state, err := sessiondb.CreateSession(workspace, "", 24)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	path := filepath.Join(home, "Documents", app.DirName, "session."+workspace+".db")
	t.Cleanup(func() { _ = os.Remove(path) })
	if err := sessiondb.CreateStreamPrompt(workspace, state.ActiveSessionID, 0, 102, "request", "start"); err != nil {
		t.Fatalf("CreateStreamPrompt: %v", err)
	}

	var events []sessiondb.AIStreamBlock
	writer := &aiStreamBlockWriter{
		workspace: workspace,
		sessionID: state.ActiveSessionID,
		runID:     102,
		emit: func(block sessiondb.AIStreamBlock) {
			events = append(events, block)
		},
	}
	ctx := withAIStreamCallback(context.Background(), &aiStreamEmitter{blocks: writer})
	tool := &einoAgentTool{delegate: &fakeAgentTool{enabled: true}}
	if _, err := tool.InvokableRun(ctx, `{"path":"main.go"}`); err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("tool-call events = %d, want open and closed events", len(events))
	}
	if events[0].Kind != sessiondb.StreamBlockToolCall || events[0].Status != "open" || events[0].Label != "fake.tool" {
		t.Fatalf("first event = %+v, want labelled open tool-call block", events[0])
	}
	if events[1].BlockID != events[0].BlockID || events[1].Status != "closed" || events[1].Delta != `{"path":"main.go"}` {
		t.Fatalf("close event = %+v, want matching closed tool-call with input", events[1])
	}
}

func TestSubagentToolActivityUsesNestedTypedBlocks(t *testing.T) {
	workspace := "stream-writer-subagent-children-test"
	state, err := sessiondb.CreateSession(workspace, "", 24)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	path := filepath.Join(home, "Documents", app.DirName, "session."+workspace+".db")
	t.Cleanup(func() { _ = os.Remove(path) })

	if err := sessiondb.CreateStreamPrompt(workspace, state.ActiveSessionID, 0, 101, "request", "start"); err != nil {
		t.Fatalf("CreateStreamPrompt: %v", err)
	}
	writer := &aiStreamBlockWriter{
		workspace: workspace,
		sessionID: state.ActiveSessionID,
		runID:     101,
	}
	subagentBlock := writer.Open(sessiondb.StreamBlockSubagent, "")
	if subagentBlock == nil {
		t.Fatal("Open subagent block returned nil")
	}
	ctx := withAIStreamCallback(context.Background(), &aiStreamEmitter{blocks: writer})
	ctx = (&AIStreamBlockEmitter{block: subagentBlock}).ChildContext(ctx)
	var response strings.Builder
	emitter := newChildAIStreamEmitter(ctx, func(text string) {
		response.WriteString(text)
		subagentBlock.Append(text)
	})
	ctx = withAIStreamCallback(ctx, emitter)

	emitter.fn("subagent answer")
	toolCallID := emitAIStreamToolBlockIDWithLabel(ctx, sessiondb.StreamBlockToolCall, `{"query":"needle"}`, "grep")
	if toolCallID == "" {
		t.Fatal("tool-call block was not emitted")
	}
	toolCtx := withAIStreamBlockParent(ctx, toolCallID)
	emitAIStreamToolBlock(toolCtx, sessiondb.StreamBlockToolOutput, "match")
	emitAIStreamToolProgress(toolCtx, "searched files")
	emitAIStreamToolBlock(toolCtx, sessiondb.StreamBlockToolError, "tool failed")
	summary := OpenAIStreamBlock(toolCtx, sessiondb.StreamBlockSummary, "")
	if summary == nil {
		t.Fatal("Open summary block returned nil")
	}
	summary.Emit("summary")
	summary.Close()
	subagentBlock.Close()

	if response.String() != "subagent answer" {
		t.Fatalf("subagent response = %q, want callback text", response.String())
	}
	if err := sessiondb.FinalizeStreamPrompt(workspace, state.ActiveSessionID, 101, 43, "finish", sessiondb.PromptStatusSuccess); err != nil {
		t.Fatalf("FinalizeStreamPrompt: %v", err)
	}
	blocks, err := sessiondb.ListStreamBlockMeta(workspace, state.ActiveSessionID, 43)
	if err != nil {
		t.Fatalf("ListStreamBlockMeta: %v", err)
	}
	if len(blocks) != 6 {
		t.Fatalf("blocks = %d, want subagent, tool-call, output, notice, error, and summary", len(blocks))
	}
	wantParents := []string{"", blocks[0].BlockID, blocks[1].BlockID, blocks[1].BlockID, blocks[1].BlockID, blocks[1].BlockID}
	for i, block := range blocks {
		if block.ParentID != wantParents[i] {
			t.Fatalf("block %q parent = %q, want %q", block.BlockID, block.ParentID, wantParents[i])
		}
	}
	wantKinds := []sessiondb.StreamBlockKind{
		sessiondb.StreamBlockToolCall,
		sessiondb.StreamBlockToolOutput,
		sessiondb.StreamBlockNotice,
		sessiondb.StreamBlockToolError,
		sessiondb.StreamBlockSummary,
	}
	for i, kind := range wantKinds {
		if blocks[i+1].Kind != kind {
			t.Fatalf("block %d kind = %q, want %q", i+1, blocks[i+1].Kind, kind)
		}
	}
	content, err := sessiondb.GetStreamBlockContent(workspace, state.ActiveSessionID, 43, blocks[0].BlockID)
	if err != nil {
		t.Fatalf("GetStreamBlockContent subagent: %v", err)
	}
	if content != "subagent answer" {
		t.Fatalf("subagent block content = %q, want callback text", content)
	}
}

# 6. Parallel sub-agent output is buffered, not streamed

Date: 2026-09-06

## Status

Superseded by 0037. Buffering mitigated interleaving for sub-agents only; 0037
gives every concurrent producer its own addressable block, so sub-agents stream
live again. The analysis below remains the definitive diagnosis of the defect.

## Context

The `delegate` and `report` tools run their jobs concurrently (`wg.Go`), and every
job was handed the *same* stream emitter obtained from the shared `ctx`.

Output in the AI panel became garbled. Investigation confirmed the write path is
**correctly synchronised** — there is no data race:

- `aiStreamEmitter` holds `e.mu` across every `flushLocked()` + `e.fn(...)`
  sequence.
- `aiSessionLogStore` guards state, file appends, and `runID`/`sequence`
  assignment under one lock.
- The frontend reorders chunks by `sequence` and drops out-of-order ones.

The corruption was **logical interleaving**, not memory unsafety. Each sub-agent's
`Run` emits a prefix (`\n> **Sub-agent NAME:** `), then quoted chunks, then a
suffix. With N concurrent jobs writing to one linear markdown stream the result
was:

```
> **Sub-agent A:** > **Sub-agent B:** <A chunk><B chunk><A chunk>…
```

Every individual write was atomic; the *blockquotes* were interleaved.
`emitReasoning` was worse still, as it accumulates into a single shared `pending`
buffer with one shared `inThinking` flag, so concurrent reasoning streams merged
mid-sentence into one blockquote.

## Decision

Give each parallel job its own buffer and emit its output as **one contiguous
block** when the job completes.

Live token-level streaming from N concurrent writers into a single linear pane is
not orderable, so it is abandoned for sub-agents. In-flight feedback is provided
by the per-job sticky notification instead.

## Consequences

- Each sub-agent's output is contiguous and attributable.
- Sub-agent output appears on completion rather than token-by-token. Accepted
  trade-off; the ordering problem has no solution that preserves both.
- Memory grows with a sub-agent's total output rather than being streamed away.
  Acceptable given the summariser interposer already caps oversized tool output.
- **Mutex-safe is not the same as correctly serialised.** When diagnosing garbled
  output, check for multiple logical writers sharing one sink before assuming a
  data race.

## Reference

- `ai/tools/subagents/subagent.go` — per-job buffer and atomic flush
- `ai/subagent/subagent.go` — `EmitStream`, prefix/suffix framing
- `ai/agent/runtime_eino.go` — `aiStreamEmitter`

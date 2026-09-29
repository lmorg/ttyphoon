# 0040 - Transient model stream disconnects use checkpoint retry

## Status

Accepted. Persisted/manual rerun behaviour is specified separately in ADR 0041.

## Context

A model request can run successfully for several minutes, complete MCP/tool
calls, and then fail while receiving a later provider stream chunk:

```text
failed to receive stream chunk: read tcp ...: connection reset by peer
```

This is a transport failure, not a tool or model decision failure. Returning it
immediately loses the remaining task context and exposes the provider/network
error as a terminal agent failure.

Replaying the exact model request is unsafe: the bounded Eino window may already
have performed tool calls, so a replay can repeat side effects.

## Decision

Retry transient stream failures through the existing structured continuation
checkpoint, not by replaying the raw request.

The outer `RunLLMWithMessageStream` loop:

1. Classifies transport failures using `syscall.ECONNRESET`, `syscall.EPIPE`,
   temporary/timeout `net.Error`s, and common stream error text such as
   `failed to receive stream chunk`, `connection reset by peer`, `connection
   closed`, `broken pipe`, and `unexpected EOF`.
2. Does not retry when the overall request context is cancelled or timed out.
3. Retries at most twice per uninterrupted request, with a 250 ms delay.
4. Serializes the current window's visible output and tool observations through
   the existing continuation checkpoint.
5. Appends that checkpoint as a continuation message, so the next bounded window
   can verify completed work instead of replaying the exact tool sequence.
6. Emits `Retrying interrupted model stream (n/2)` to the AI panel.

Rate-limit responses use the same checkpoint continuation rather than failing the
prompt or replaying completed tool work. HTTP 429 is detected from the OpenAI
adapter's typed status and conservative message markers (`status code: 429`,
`429 Too Many Requests`, or `Too Many Requests`) for wrappers such as Eino's
`NodeRunError`. Rate limits retry at most three times. A provider `Retry-After`
header, in milliseconds (`Retry-After-Ms`), seconds, or HTTP-date form, is
honoured exactly when it survives SDK wrapping; otherwise delays use capped
exponential backoff of 1, 2, then 4 seconds. Waiting remains interruptible by
the global request context.

The retry counters reset after a successful bounded window. Other non-transport
errors still return immediately, and max-step errors continue through the
existing user continuation flow from ADR 0003.

## Consequences

- Short-lived provider, proxy, or network resets recover automatically.
- Tool side effects are not blindly replayed.
- A persistent outage still fails after two retries rather than hanging or
  retrying indefinitely.
- A persistent rate limit fails after three retries. OpenRouter's OpenAI adapter
  retains status 429 but discards response headers, so it normally uses the
  exponential fallback; providers that expose their HTTP response use
  `Retry-After`.
- The retry delay is included in the global request timeout from ADR 0004.
- The checkpoint remains a best-effort summary: reasoning and incomplete
  provider scratchpad state are not reconstructed as structured model state.
- Error classification includes message matching because the provider SDK does
  not expose one stable typed error for every transport failure. Keep the list
  conservative and expand it only with a regression test.

## Reference

- `ai/agent/runtime_eino.go` — `RunLLMWithMessageStream`,
  `isTransientStreamError`, `waitForTransientStreamRetry`
- `ai/agent/runtime_eino_test.go` — retry and classification coverage
- ADR 0003 — continuation checkpoint flow
- ADR 0004 — global request timeout

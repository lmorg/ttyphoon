# 4. AI request timeout is global configuration

Date: 2026-09-06

## Status

Accepted

## Context

The agent request deadline was hard-coded in `ai/ui.go`:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
```

When the AI settings panel began displaying execution limits (ADR 0011), the same
value was hard-coded a second time as the string `"5m"`. Two independent literals
describing one behaviour will drift.

## Decision

Introduce `AI.RequestTimeout` as a YAML duration string, parsed through a single
accessor:

```go
func (ai *AiT) RequestTimeoutDuration() time.Duration
```

The accessor falls back to 5 minutes when the value is empty, unparseable, or
non-positive, so a malformed config degrades rather than producing a zero-length
(instantly-expiring) timeout.

Both the request context and the settings display read from this accessor. There
must be no other source for this value.

## Consequences

- Users can extend the deadline for long-running agent work.
- The displayed limit is guaranteed to match runtime behaviour.
- The timeout covers the **entire** run including all continuation windows
  (ADR 0003), so raising `MaxContinuations` may also require raising this.
- Other timeouts remain deliberately independent and separately scoped:
  image generation (5m), MCP OAuth callback (2s), Anthropic max output tokens
  (8192).

## Reference

- `config/config.go` — `AiT.RequestTimeout`, `RequestTimeoutDuration()`
- `config/defaults.yaml` — `AI.RequestTimeout`
- `ai/ui.go` — request context construction
- `frontend.go` — `GetAIExecutionLimits`

# 11. Report harness limits; do not invent provider limits

Date: 2026-09-06

## Status

Accepted

## Context

"What are the max steps for this model?" is a natural question with a misleading
premise. Providers do not expose a maximum-iterations value. They publish:

- context-window size
- maximum output tokens
- rate limits
- tool/function-calling support

The iteration limit is a property of the **agent harness**, not the model. In
ttyphoon it is `config.Config.Ai.MaxIterations`, passed to Eino as `MaxStep`.

Conflating the two leads to wrong conclusions — for example assuming a 128k
context window implies a large step budget, when the run still stops at 10 steps.

## Decision

The AI settings panel reports execution limits as distinct, separately-sourced
fields, and explicitly reports `Unknown` where no reliable value exists:

| Field | Source |
|---|---|
| Agent step budget | `AI.MaxIterations` |
| Provider max iterations | `Unknown` — not a concept providers expose |
| Model context window | `Unknown` until model metadata is integrated |
| Model max output tokens | `Unknown` unless configured per provider |
| Request timeout | `AI.RequestTimeout` (ADR 0004) |

`Unknown` is preferred over a plausible guess. A wrong number here would be used
to reason about failures and would mislead.

## Consequences

- Users can see which limit actually governs a run.
- Values ttyphoon controls are exact, because they are read from the same
  accessors the runtime uses.
- Populating the model fields requires integrating provider/model metadata
  (OpenRouter exposes context and output-token limits; the Anthropic output cap is
  already hard-coded as `einoAnthropicMaxTokens = 8192`).

## Reference

- `frontend.go` — `GetAIExecutionLimits`
- `frontend/src/notes.js` — `renderAIExecutionLimits`
- `ai/agent/runtime_eino.go` — `MaxStep`, `einoAnthropicMaxTokens`

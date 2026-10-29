# 81. Per-prompt effort guides investigation and user-feedback cadence

## Status

Accepted and implemented on 2026-10-29.

## Context

AI requests use several entry points but share the same one-shot agent harness.
The harness has an `askUser` tool for blocking clarification, which should be
used only when missing required information cannot be resolved safely. Users
need a way to ask for a quicker focused pass or a more thorough investigation
without implying a guaranteed duration or provider-specific model setting.
Requests span software, documentation, operations, research, and other work, so
effort guidance must not assume a coding task.

## Decision

- Add a per-request `Effort` choice with `Quick`, `Balanced`, and `Thorough`
  options. `Quick` is the default, including when a prompt has no explicit
  effort value.
- Include the choice in every prompt inputbox that invokes AI, including general
  prompts, skills, URI prompts, and document or image explanations.
- Share one inputbox draft across AI entry points within a tmux workspace name;
  submitted history remains scoped independently by its existing inputbox key.
- Embed the mode-specific instructions as separate text files and append the
  selected text to the request's system prompt. Quick keeps
  work focused on the requested outcome and raises consequential unresolved
  preferences earlier; Thorough explores relevant alternatives, edge cases,
  dependencies, and downstream effects before asking. Instructions use
  domain-neutral language. All modes preserve the `askUser` contract: don't
  interrupt for routine uncertainty; ask only when the answer materially affects
  the result or safe next step.
- Treat effort as portable behavioral guidance, not a hard time budget, an
  iteration limit, or a provider API parameter. Provider-specific reasoning
  controls can be added separately when they can be mapped reliably.

## Consequences

- Users can choose the desired balance for each prompt without changing global
  model or tool settings.
- The selected guidance is captured in the full prompt transcript.
- Actual elapsed time and reasoning depth remain model/provider dependent.

## Reference

- `ai/ui.go` - shared effort input and prompt entry points
- `window/backend/renderer_webkit/ai.go` - toolbar and skill prompt entry points
- `ai/prompts/prompts.go` - effort-specific system instructions
- `ai/prompts/effort_quick.txt`, `effort_balanced.txt`, and
  `effort_thorough.txt` - editable effort guidance text
- `ai/prompts/prompts_test.go` - selected effort guidance tests
- ADR 0008 - `askUser` is for blocking clarification
# 13. Adding a Wails binding requires three coordinated edits

Date: 2026-09-06

## Status

Accepted

## Context

Adding `ResolveAIUserQuestion` produced a runtime failure that looked like a
backend bug:

```
[Error] ReferenceError: Can't find variable: ResolveAIUserQuestion
  (anonymous function) — notes.js:12518
```

The Go method existed and the generated binding files contained it. The failure
was that `notes.js` had not added the symbol to its `import { … } from
'../wailsjs/go/main/WApp'` list. The symbol was simply not in scope.

The failure mode is misleading: the error surfaces inside a `catch` block as a
generic "Failed to …" notification, pointing suspicion at the Go implementation.

## Decision

Every new frontend-callable method requires all of:

1. An exported method on the single `WApp` service in `frontend.go`.
2. Regenerate bindings with `wails generate bindings`. Generated service code
  lives under `frontend/bindings`; do not edit it by hand.
3. Import the method through `frontend/src/wails-bindings.js`. This keeps
  application modules independent of the generator's service directory layout.
4. Update the matching Vitest mock of `wails-bindings.js`, or tests touching
  that code path will fail.

For bindings called from event handlers, guard the call so a stale build degrades
instead of throwing:

```js
if (typeof SetAIPanelLive === 'function') { … }
```

## Consequences

- A known checklist replaces guesswork when a binding "does not exist".
- Diagnosis heuristic: `ReferenceError` in the browser console means **frontend
  wiring**; an error returned from a resolved promise means **backend logic**.
- A stale running app can also cause this. Rebuilding/restarting is the first
  thing to rule out before changing code.

## Reference

- `frontend.go` — exported `WApp` methods
- `frontend/bindings/github.com/lmorg/ttyphoon/wapp.js` — generated service
- `frontend/src/wails-bindings.js` — stable frontend import boundary
- `frontend/src/notes.js` — frontend binding imports
- `frontend/src/notes.test.js` — mock of the stable import boundary

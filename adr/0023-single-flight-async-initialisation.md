# 0023 — Single-flight async initialisation

## Status

Accepted.

## Context

The Notes window logged a Monaco boot failure:

```
[Error] Element already has context attribute
    ScopedContextKeyService
    StandaloneCodeEditor2
    createMonacoAdapter (monaco_adapter.js:619)
```

followed by a cascade of `[createInstance] _aNN depends on UNKNOWN service
ICodeLensCache / IInlayHintsCache / ISuggestMemories / treeViewsDndService`.

`Element already has context attribute` is Monaco's error for calling
`monaco.editor.create()` on a container that already hosts an editor. The
`UNKNOWN service` errors are downstream fallout from that failed instantiation,
not independent problems.

The guard looked correct:

```js
async function ensureMonacoMainEditor() {
    if (!state.useMonacoEditor || !elements.monacoEditor || monacoMainEditor) {
        return;
    }
    monacoMainEditor = await createMonacoAdapter(elements.monacoEditor, { ... });
}
```

But `monacoMainEditor` is assigned **after** an `await`. Two calls arriving
before the first resolves both pass the guard and both call
`monaco.editor.create()` on the same element. The only caller is
`ensureMonacoVisibleAndLaidOut`, invoked fire-and-forget (`void`) from the view
update that runs whenever the editor wrap becomes visible — so overlapping
invocations are routine (file load plus tab switch in the same frame).

The consequence is worse than the noise suggests. Whichever creation loses the
race rejects, and because the caller is `void`-invoked the rejection is an
unhandled promise — invisible unless the console is open. If both fail,
`monacoMainEditor` stays `null` while a partially constructed Monaco DOM is still
on screen, so `isMonacoActive()` returns false and every Monaco-dependent path
silently degrades to the hidden textarea (see ADR 0021).

## Decision

Share one in-flight creation between concurrent callers:

```js
if (!monacoMainEditorInit) {
    monacoMainEditorInit = createMonacoMainEditor().finally(() => {
        monacoMainEditorInit = null;
    });
}
await monacoMainEditorInit;
```

The creation body moved to `createMonacoMainEditor`; `ensureMonacoMainEditor` is
now just the single-flight gate. Clearing the handle in `finally` keeps failures
retryable rather than latching a permanently broken editor.

The awaited result is wrapped in `try/catch` that logs. A failed editor boot
must not be an unhandled rejection — that is precisely what made this hard to
attribute.

## Consequences

- One editor per container; the context-attribute and UNKNOWN-service errors go
  away together.
- Monaco boot failure is now reported instead of silent.
- Not covered by tests: Monaco is disabled under jsdom, so neither the race nor
  the fix is reachable from Vitest (ADR 0014).

## Lessons

- **A truthy guard before an `await` does not serialise anything.** The window
  between the check and the assignment is exactly as long as the async work it
  guards. Memoise the *promise*, not the result.
- **`void asyncFn()` converts failures into invisible ones.** Fire-and-forget
  callers need an explicit catch, or the first symptom is unrelated behaviour
  breaking somewhere far away.
- **Read an error cascade top-down.** Four `UNKNOWN service` errors looked like a
  bundling or dependency-injection problem; all four were consequences of the
  single create failure logged above them.

## Reference

- `frontend/src/notes.js` — `ensureMonacoMainEditor`, `createMonacoMainEditor`,
  `ensureMonacoVisibleAndLaidOut`
- `frontend/src/monaco_adapter.js` — `createMonacoAdapter`
- ADR 0021 — silent degradation when `isMonacoActive()` is false

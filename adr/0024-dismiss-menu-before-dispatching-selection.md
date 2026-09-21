# 0024 — Dismiss a menu before dispatching its selection

## Status

Accepted.

## Context

`Go to workspace symbol` opened no menu at all. Nothing was logged, so there was
no exception — the code ran to completion and the menu still never appeared.

`selectMenuItem` dispatched the selection and *then* dismissed the menu:

```js
function selectMenuItem(item) {
    _menuOperationInProgress = true;
    if (activeListMenuId !== null) {
        menuSelect(activeListMenuId, item.index);   // runs the handler
    }
    hideListMenu(false);                            // then tears down
}
```

`hideListMenu` is not scoped to a particular menu. It resets the shared listbox:
`activeListMenuId = null`, clears `listItems` / `filteredItems`, unsets
`showSearch` and `dynamicQuery`, and animates the single `listRoot` to
`display: none`.

So any handler that opens another menu **synchronously** has that new menu
destroyed the moment it returns.

This was latent for as long as the LSP submenus existed, but hidden: every one of
them awaited a language-server round trip before calling `showLocalMenu`, so the
new menu opened a microtask *after* the teardown and survived. Moving workspace
symbol search server-side (ADR 0022) removed its only pre-menu `await` in the
common case — the document is already attached, so nothing suspends — and the
latent bug became reproducible.

`goToCurrentLspSymbol` still awaits `NotesLspDocumentSymbols`, which is why one
symbol command worked and its sibling did not.

## Decision

Dismiss first, dispatch second. `activeListMenuId` is captured before the hide,
because `hideListMenu` nulls it:

```js
const menuId = activeListMenuId;
hideListMenu(false);
if (menuId !== null) {
    menuSelect(menuId, item.index);
}
```

Applied to both dispatch paths — mouse selection and the `Enter` key handler,
which had the same ordering.

`menuSelect` also restored focus to whatever was focused before the menu opened,
which would steal focus from a newly opened menu's search field. `showLocalMenu`
re-assigns `_localMenuReturnFocus`, so a non-null value after the handler returns
is a reliable signal that a new menu took ownership:

```js
if (returnTo && !_localMenuReturnFocus) returnTo.focus();
```

## Consequences

- Handlers may open a submenu synchronously or asynchronously; both work.
- Selection callbacks now run against a already-dismissed menu. They must not
  assume menu state is still live — none did, since `hideListMenu` ran
  immediately afterwards anyway.

## Lessons

- **An `await` can be load-bearing.** The original code was only correct because
  every handler happened to suspend first. Removing an unrelated network call
  broke a menu — timing was doing the work that ordering should have done.
- **"Nothing happened, nothing logged" points at teardown, not failure.** No
  exception meant the open path ran fine; something undid it afterwards.
- **Verify a regression test actually regresses.** Restoring the old ordering
  turned the new test red with `expected 'none' to be 'block'`, confirming it
  pins the real defect rather than passing incidentally.

## Reference

- `frontend/src/popup_menu.js` — `selectMenuItem`, `menuSelect`, `hideListMenu`,
  window `keydown` Enter handler
- `frontend/src/popup_menu.test.js` — `keeps a menu opened synchronously from
  another menu selection`
- ADR 0022 — removing the pre-menu `await` that had been masking this

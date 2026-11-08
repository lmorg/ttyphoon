# 0092 - Organize Wails code by window and feature

Date: 2026-11-05

## Status

Accepted and implemented.

Supersedes [ADR 0085](0085-wails-v3-migration.md).

## Context

ADR 0085 deliberately prioritized porting existing functionality to Wails v3
over restructuring it. Keeping the large `WApp` facade and window code at the
repository root reduced migration risk, but was a transitional arrangement,
not the desired long-term layout.

The main window now shares the application with browser windows opened from
Notes links. Wails v3 services can live in ordinary Go packages, and a service's
methods can be split across feature files without creating separate services.

## Decision

- Put main-window Go code in `window/frontend/main`, using `package mainwindow`.
  The directory name describes its role; `package main` would make it an
  executable package rather than an importable library.
- Put browser-window Go code in `window/frontend/browser`, using `package
  browser`. It owns web URL validation, window creation, focus, and the
  configured always-on-top behavior.
- Split the main-window package by feature: for example, `service.go` for
  `WApp` state and construction, `window.go` for lifecycle, `runtime.go` for
  its desktop runtime adapter, and `lsp.go`, `terminal.go`, `notes.go`,
  `notes_index.go`, `tables.go`, `git.go`, `ai.go`, and `hyperlinks.go` for the
  corresponding bindings and helpers. File boundaries should follow cohesive
  responsibilities rather than arbitrary size limits.
- Retain one `WApp` service initially. File organization does not require
  splitting shared state or introducing one service per feature. Browser
  windows do not need a separate service until they have their own bindings.
- Keep root code limited to executable startup and asset embedding. Put
  application composition, service registration, and the application-wide
  menu in `window/wails.go` (`package window`). Embed frontend assets and the
  icon in `main.go` and pass them to `window.StartWails`:
  `go:embed` cannot reference parent directories.
- Keep dependencies one-way: the main-window adapter may use the browser
  package, but the browser package must not depend on the main-window package.
  Keep core logic in existing backend packages and preserve the
  `types.DesktopRuntime` / `utils/wailsruntime` boundary; do not pass
  `*application.App` into core backend code.
- Move related tests with their implementations. Regenerate Wails bindings
  after moving the service and update the stable
  `frontend/src/wails-bindings.js` import boundary and affected mocks. Do not
  hand-edit generated files or retain root forwarding methods solely to
  preserve old generated paths.

### Behavior to preserve

Superseding ADR 0085 removes its migration-era layout constraint, not the
application behavior it preserved:

- Keep existing frontend event names and payloads, frontend-ready startup, and
  shared-state synchronization required by ADR 0010.
- Preserve maximized main-window startup, the hidden macOS title bar,
  hide-on-close behavior, the notes asset handler, and the product identifier
  `com.lmorg.ttyphoon`.
- Implement hide-on-close with a Wails `RegisterHook` for `WindowClosing`.
  Hooks can cancel destruction before event listeners run; the old
  asynchronous `OnWindowEvent` listener raced Wails' own closing listener.
- Showing the main window also focuses it, including the show branch of the
  F12 global visibility toggle. Hiding the window does not request focus.
- Keep the standard Wails About dialog and native browser-context-menu
  suppression without interfering with application menus.
- Set an explicit application menu containing only the app-name submenu and
  Wails' About role. Do not include Wails' default Edit, Window, Services, Hide,
  or Quit entries; in-app context menus remain unchanged.
- Keep `ttyphoon://` interactions inside the application without registering
  the scheme with the operating system.
- Preserve browser-window link gestures, focus, and always-on-top behavior
  from ADR 0091.
- Keep the Wails v3 build and binding-generation workflow. Version-specific
  migration notes in ADR 0085 are historical, not permanent dependency pins.

## Consequences

- Root-level Wails bindings, feature helpers, and window-specific runtime code
  are legacy placement, not precedent for new code. New functionality belongs
  in the window and feature package that owns it.
- When future work finds code missed by this refactor, assess and move the
  directly affected legacy code into this layout rather than extending the
  root facade or adding compatibility layers around it. This does not justify
  unrelated rewrites.
- Preserve existing behavior during relocation. Package changes must not
  duplicate service state, weaken locking, change window lifecycle semantics,
  or silently break generated bindings.
- Validate relocation with binding regeneration, affected Go and frontend
  tests/builds, and native checks of both window lifecycles.

## Verification

- Regenerated one Wails service with all 211 callable methods, using the new
  package path. All 203 original `frontend.go` service methods are retained;
  index and table methods remain on the same service.
- Relocated Go tests, the asset-handler routing regression, and
  `go build ./...` pass. The production frontend build passes.
- All 350 relocated declarations from the original tracked files, including
  their existing tests, are token-identical; application composition and
  shutdown wiring were moved separately, and the closing listener was
  intentionally changed to a cancellation hook.
- The frontend suite passes 677 of 679 tests. Two existing print-style
  assertions in the unchanged `notes-print.test.js` expect point subtraction
  while the unchanged stylesheet uses print scaling; these are unrelated to
  the relocation and were not altered.
- An isolated native macOS smoke check (without starting terminals, tmux,
  watchers, or global hotkeys) confirms main-window visibility and maximization,
  browser visibility and focus with always-on-top configured, independent
  browser closing, main-window hide-on-close, and showing the main window again.

## References

- [ADR 0085: Wails v3 migration (superseded)](0085-wails-v3-migration.md)
- [ADR 0010: shared-state synchronization](0010-wails-shared-state-synchronisation.md)
- [ADR 0013: Wails binding checklist](0013-wails-binding-checklist.md)
- [ADR 0091: Notes web link windows](0091-notes-markdown-middle-click-opens-wails-window.md)
- `main.go` - executable startup and embedded assets
- `window/wails.go` - application composition, service registration, and menu
- `window/frontend/main` - main-window service, feature bindings, adapter,
  lifecycle, and relocated tests
- `window/frontend/browser` - browser-window creation and URL validation
- `frontend/src/wails-bindings.js` - stable frontend binding import boundary

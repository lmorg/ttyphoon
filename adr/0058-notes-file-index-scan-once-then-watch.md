# 0058 - Notes file list is a Go-owned index: scan once, then watch

## Status

Accepted. Stages 1-3 were implemented on 2026-10-07: cached snapshots,
background scans, startup prewarming, refresh-state UI, active-root directory
watching, and versioned deltas. Persistent snapshots and the opt-in "hide
gitignored files" setting are not needed at this time; they remain recorded
for future reconsideration, not as outstanding implementation work.

## Context

### How the file list worked before stage 1

- `WApp.ListFiles` (`frontend.go`) calls `notes.ListFiles` (`utils/notes/walk.go`),
  which resolves the project root from the **active tile's `Pwd()` at call
  time**, then does a full synchronous `filepath.WalkDir` of that root (capped at
  `MAX_FILES`) plus `$GLOBAL` and `$NOTES/<group>`.
- As a side effect, `ListFiles` overwrites `a.projRoot`, `a.usrNotesDir`,
  `a.globalNotes` and `a.groupName`. Grep, history and other features read
  those fields, so "which project am I in" is only updated when the list is
  rescanned.
- The frontend's only trigger is the `notesUpdate` event. On receipt it calls
  `CancelNotesListFiles()` then `refreshFiles()` → `ListFiles()`.
- `notesUpdate` is emitted by `RefreshNotes()` from three places:
  - `Term.SetFocus(true)` (`virtualterm/term.go`) — every terminal focus/click.
  - `_osc7UpdatePath` (`virtualterm/osc_functions.go`) — every OSC 7 cwd report.
    Most shell integrations emit OSC 7 on **every prompt**, not only on `cd`.
  - `RefreshWindowList` (`renderer_webkit/interface.go`) — tmux window changes.
- `NS_NOTESW_FILES` is not a file-list cache. `walk.go` reads it into
  `ulf.Files` and immediately writes the same value back; nothing ever writes
  scan results into it. It only refreshes its own TTL.
- `utils/file/watcher` is a singleton `fsnotify` watcher for the **open
  document only**. `Watch()` removes every existing watch first, so it cannot
  be reused for directory indexing as-is.

### Why the reported symptoms happen

1. **`$PROJECT` sometimes empty after switching window/workspace.**
   - Root resolution races the tmux switch: `RefreshWindowList` can fire before
     the new active pane's `Pwd()` is known, so the walk targets the old (or an
     empty) root. A later focus click re-resolves it — hence "click the
     terminal to fix it".
   - A cancelled walk is returned as if complete. `WalkDir` stops with
     `context.Canceled`, but the partial `projectFiles` slice is still returned
     and rendered as authoritative. Rapid `notesUpdate` bursts (window change +
     focus + OSC 7) make cancellation routine.
   - `refreshFiles()` has no sequence guard, so an older response can land
     after a newer one and overwrite it.
2. **New files (e.g. from AI tools) don't appear.** Nothing observes the
   filesystem; the list only changes when an unrelated event triggers a rescan.
3. **Unnecessary disk I/O.** Every focus click and (typically) every shell
  prompt performs a full tree walk of up to `MAX_FILES` entries, even when neither the
   root nor its contents changed.
4. **No useful cache.** There is no cache of scan results, so switching back to
   a workspace always pays for a full rescan.

The underlying design problem is that two different concerns are coupled into
one expensive call:

- **Context:** which roots are active (cheap — a string comparison).
- **Content:** what files exist under those roots (expensive — a tree walk).

## Decision

Separate context from content, and make content a long-lived, Go-owned index
that is populated in the background, reconciled on activation, and kept current
by filesystem events while its root is active.

### 1. Go owns a `FileIndex` per root, not per workspace

- Key indexes by **absolute root path**. Multiple tmux windows frequently share
  a repository, and `$GLOBAL` / `$NOTES/<group>` are just additional roots.
  This gives cache reuse across workspaces for free.
- Each index holds the file set, a monotonically increasing `version`, and a
  state: `scanning`, `refreshing`, `ready`, `truncated` (hit the file cap), or
  `error`, plus whether its last published snapshot needs reconciliation.
- The initial scan runs in a goroutine and **never blocks** the Wails call
  that asked for it. Callers get the current snapshot (possibly empty with
  state `scanning`) immediately; completion is announced by event.
- Keep a bounded LRU of cached snapshots separately from watcher ownership.
  Only roots required by the active workspace retain directory watches;
  deactivation releases those watches immediately but retains the snapshot.
  A root shared with the newly active workspace remains watched.
- On reactivation of an unwatched root, return its cached snapshot immediately
  and reconcile it in the background. The cache can be stale until that scan
  completes; do not clear the list while refreshing.
- At application startup, prepopulate roots for known workspaces using a
  bounded background queue, deduplicated by root. Prioritize the active roots;
  do not delay application startup or install directory watches for inactive
  roots. Workspaces discovered later are populated on first activation.

### Scan ownership and publication

- Ordinary activation and reconciliation scans use a cancellable Go context.
  Cancel the scan when its root is no longer required by the active workspace.
  Walks check cancellation between entries; cancellation cannot necessarily
  interrupt an OS filesystem call already in progress.
- Startup prewarm scans are the exception: they use an application-lifetime
  context and continue when their workspace becomes inactive, but still cancel
  on shutdown. An activation adopts an existing startup scan rather than
  launching a duplicate; ensure changes since it started are reconciled before
  declaring the active index current.
- Build scan results privately. Under the index lock, verify the context is
  not cancelled and the scan generation is still current before atomically
  publishing the snapshot and incrementing its version. Cancellation and
  generation invalidation use the same synchronization boundary. Cancelled,
  failed, or superseded scans never replace the last complete snapshot or emit
  a successful-completion event.
- Keep at most one scan worker per root. Rapid deactivation/reactivation
  cancels the old ordinary scan, waits for its worker to exit, then starts one
  fresh scan if that root is still active. Coalesce repeated requests; do not
  reuse partial results from a cancelled scan. Retain a stale snapshot on
  cancellation and retry on activation; surface scan errors without losing it.

### 2. Context changes are cheap and only emit when something changed

- Resolve the active context (project root, notes dir, group name) in Go when
  focus, OSC 7, or the tmux window changes. Emit `notesContextChanged` **only
  if the resolved roots differ** from the last emitted value. Focus clicks and
  per-prompt OSC 7 inside the same project then cost a path comparison, not a
  walk.
- `a.projRoot` and friends are updated by context resolution, not as a side
  effect of listing, so grep/history see the right root immediately.
- When the tmux pane `Pwd` is not yet known after a window switch, resolve it
  from tmux (`pane_current_path`) rather than emitting with a stale value.

### 3. Filesystem watching keeps the index current

- Watch each active root and apply create/remove/rename events to the index
  incrementally. A created directory triggers a background scan of just that
  subtree.
- Attach watches before activation scans and collect change notifications while
  scanning. Reconcile affected paths before publishing the refreshed snapshot,
  so changes during the scan are not lost. Recursive watch installation must
  also handle directories created during setup; a dirty root or overflow
  requires another reconciliation before it can be considered current.
- Inactive snapshots are not maintained by directory watches. A startup scan
  that completes while inactive may populate the cache, but its result still
  requires reconciliation when that root becomes active.
- **Coalesce** events (≈150–300 ms) and emit one `notesFilesChanged` per root
  per window, so `git checkout`, `npm install` or a build doesn't flood the
  frontend.
- Apply the same `ExcludeDirectories` and system-file filters used by the scan
  to events, so ignored directories (e.g. `node_modules`) aren't watched.
- On watcher overflow or loss (inotify queue overflow, FSEvents
  "must rescan"), fall back to a background rescan of the affected root.
- Go-side writes we already know about (Notes saves, AI file tools,
  rename/delete bindings) should update the index directly as a write-through
  hint. That makes our own changes visible instantly and doesn't depend on
  watcher latency.

### 4. Frontend consumes snapshots and versioned deltas

- On `notesContextChanged`, fetch snapshots for the active roots; these come
  from memory, so a workspace switch is instant after the first visit.
- On `notesFilesChanged { root, version, added, removed }`, apply the delta if
  `version` follows the one held; otherwise (gap or unknown root) re-fetch the
  snapshot. This avoids re-sending up to `MAX_FILES` paths on every change.
- Ignore responses for roots that are no longer active, and guard requests with
  a sequence number so stale responses never overwrite newer state.
- Render `scanning`, `refreshing`, and `truncated` states explicitly instead of
  showing an empty or silently clipped list. Reuse the existing CSS spinner
  styling/animation for a small refreshing icon beside the file-list heading;
  keep the cached rows visible and usable while refreshing. Clear the indicator
  on completion, cancellation, or error; preserve error/truncation feedback.

### 5. Remove the existing triggers

- `SetFocus` and OSC 7 stop calling `RefreshNotes()` directly; they feed context
  resolution instead.
- `CancelNotesListFiles` and the cancel-then-rescan pattern go away; scans are
  owned and deduplicated by the index (one in-flight scan per root, per
  ADR 0023's single-flight pattern).
- Delete the inert `NS_NOTESW_FILES` read/write.

## Review of the original proposal

The suggested direction — one initial background scan, a filesystem watch, a Go
cache that can be handed to the frontend on workspace switch, and a watch hook
that pushes updates — is sound and is the core of this decision. The
refinements above address issues that idea alone would hit:

- **Index by root, not workspace.** Avoids duplicate scans and watchers when
  windows share a repo.
- **Decouple context from content.** A cache alone doesn't fix the
  wrong-root-after-switch bug; that's a context-resolution race.
- **Recursive watching is platform-specific.** `fsnotify` is not recursive.
  - macOS: its `kqueue` backend opens a file descriptor per watched file and
    directory, which can exhaust the default descriptor limit on a `MAX_FILES`-entry
    tree. Prefer **FSEvents** (recursive, kernel-coalesced, one stream per root),
    e.g. via `github.com/fsnotify/fsevents`.
  - Linux: `inotify` needs a watch per directory and is bounded by
    `fs.inotify.max_user_watches`; handle `ENOSPC` by falling back to periodic
    rescans for that root.
  - Hide this behind a small `dirwatch` interface with per-OS backends and a
    polling fallback.
- **Coalescing and overflow recovery are required.** Without them, bulk
  operations either flood the UI or silently desynchronise the index.
- **Write-through for our own writes.** Faster and more reliable than waiting
  for the watcher to report changes we caused.
- **Watch for runaway roots.** `DirProjectRoot` falls back to `~/Documents`
  when the cwd is `$HOME`. Recursively watching that can be large; respect the
  file cap and don't watch beyond a truncated scan.

## Alternatives considered

- **Keep rescanning, but debounce and skip when the root is unchanged.** Fixes
  most wasted I/O cheaply, but still doesn't show AI-created files without a
  trigger and still walks on every real switch. Worth doing as a first step,
  not as the end state.
- **Poll directory mtimes.** Portable, but cost scales with directory count
  and changes appear late. Kept only as the fallback backend.
- **`git ls-files --cached --others --exclude-standard` for the initial scan.**
  Fast, but it changes what the list contains: `--exclude-standard` drops every
  `.gitignore`d file, so `.env`, local scratch notes and ignored docs would
  vanish from `$PROJECT`. Adding them back needs a second
  `--others --ignored` pass, which also returns `node_modules`, build output,
  etc. — effectively a full walk again. It only works inside git repositories
  and doesn't remove the need for watching. An opt-in "hide gitignored files"
  mode is **not needed at this time**. Keep it recorded for reconsideration,
  not as a replacement for the walk or planned implementation work.
- **Persist snapshots to disk for instant cold start.** **Not needed at this
  time.** Retained for future reconsideration: a persisted snapshot could be
  shown as stale while a background scan reconciles it, but an incorrect list
  can be worse than a short spinner. The current cache remains in memory only.
- **Watch from the frontend.** Not possible from the webview, and it would
  duplicate the Go-side exclusion logic.

## Platform scope

Only macOS and Linux are supported by this proposal. Windows and all other
platforms are not supported and require no watcher backend or compatibility
work. Polling is a recovery fallback for watcher failures or unsuitable
filesystems on macOS and Linux, not a way to support additional platforms.

## Phasing

1. **Implemented: correctness and I/O quick wins (no watcher).** Context is
  resolved from the active tmux pane, whose cwd is populated from
  `pane_current_path`. Unknown cwd does not fall back to the application's
  working directory. Resolution is cached until cwd or group changes.
  `RefreshNotes` now emits `notesContextChanged` only when resolved roots or
  group change; unchanged focus and prompt reports do not trigger scans.
  Backend workspace fields update independently of scanning through locked
  accessors. `GetNotesContext` and `ListFilesForContext` keep title, project,
  and scan roots together; cancelled or superseded scans reject rather than
  publish partial lists. Frontend request generations reject late results,
  including late cache/filter responses. The inert `NS_NOTESW_FILES` read/write
  is removed. The `MAX_FILES` entry cap remains. Compatibility methods `ListFiles`
  and `CancelNotesListFiles` remain, but Notes no longer uses the old
  cancel-then-rescan event chain.
2. **Implemented: in-memory per-root index.** `notes.FileIndex` stores immutable
  published snapshots by cleaned absolute root path. The default LRU budget is
  16 roots, excluding roots that are active or still have workers. A priority
  queue runs at most two scans concurrently and favours active roots. Startup
  prewarming covers known tmux workspace directories and shares workers when
  roots overlap; it does not block startup or install directory watches.
  Ordinary scans cancel on deactivation, while startup scans finish under the
  application lifetime. Rapid reactivation waits for the cancelled root worker
  to exit before starting its replacement. Shutdown cancels and joins workers.
  `GetNotesFilesSnapshot` returns cached rows without walking the filesystem;
  `notesFileIndexChanged` carries root, content version, and state revision,
  triggering coalesced snapshot reads rather than new scans. Content versions
  increment only on publication; state revisions also track refresh/error
  transitions. Errors and cancellation retain the previous published files.
  Notes reuses its existing spinner styling beside the workspace title while
  keeping cached rows usable, and exposes truncation/errors through a warning
  icon. All three sections share the index, but `$GLOBAL` and `$NOTES` retain
  their existing immediate-child display policy; `$PROJECT` remains recursive.
  The filesystem-entry cap remains `MAX_FILES`, including directories. Full snapshots
  are transferred in this stage; added/removed deltas belong to stage 3.
3. **Implemented: active-root watch backends and deltas.** macOS uses
  `github.com/fsnotify/fsevents`; Linux uses recursive directory-only fsnotify
  watches. Watches attach before activation scans and are cancelled when a
  root becomes inactive; shared active roots remain subscribed. Existing macOS
  roots subscribe only to themselves, while absent roots subscribe to their
  nearest existing ancestor until root creation is observed. Linux also
  watches an ancestor to detect absent-root creation. Excluded directories and
  system files are filtered before indexing and Linux watch installation.
  Changes coalesce for 200 ms with at most 256 pending paths per root. Ordinary
  events stat affected paths and scan created/renamed subtrees, not the entire
  project. Uncertain events, overflow, large batches, truncated roots, and
  entry-budget limits trigger capped full reconciliation. Events arriving
  during an active scan force reconciliation before its result is current.
  Watch setup/resource failures and root loss fall back to five-second
  background reconciliation on supported platforms; polling does not
  supersede an already-running slow scan. The `MAX_FILES` entry cap remains in effect
  for scans and incremental updates. macOS resolves physical FSEvents paths
  back to the original index root, including `/var` versus `/private/var`.
  Successful Notes text/binary saves, rename/delete operations, and AI write
  and atomic-edit helpers send coalesced write-through path hints independent
  of native notification timing. Content versions advance only when membership
  changes; state revisions are monotonic across LRU eviction. Events carry a
  base version, added/removed paths, and state metadata. Notes applies matching
  deltas directly, ignores old revisions, and fetches a snapshot on a gap or
  during an in-flight workspace refresh. Known frontend writes no longer
  request redundant full scans. The existing open-document watcher remains
  unchanged. Native macOS tests cover create/rename/remove, new directories,
  absent roots, cancellation, and stream shutdown; Linux watcher/index tests
  have been cross-compiled, but require native Linux execution for runtime
  certification.
4. **Not needed at this time:** persistent snapshots and an opt-in "hide
  gitignored files" mode. Both are retained as design options in case the
  decision changes; neither is outstanding implementation work.

## Consequences

- Workspace switches become a memory read after the first visit; terminal
  focus and shell prompts stop causing disk walks.
- External and AI-created files now appear without terminal focus or a manual
  refresh, subject to the display filters and entry cap. Native updates use the
  coalescing window; polling recovery can take five seconds plus scan time.
- New concurrency surface in Go: the index, watcher goroutines and Wails calls
  share state and must follow ADR 0010 (explicit synchronisation).
- Watch resources (descriptors, inotify watches) become a managed budget;
  active-root ownership bounds watches independently of cached snapshot count.
- Returning to an unwatched workspace requires a background reconciliation.
  Rapid switching can repeatedly cancel scans, leaving cached data stale until
  the user remains in a workspace long enough for reconciliation to complete.
  Startup scans are exempt from deactivation cancellation so initial caches
  can still be populated despite switching.
- Two watchers will coexist: the existing open-document watcher (content
  changes, self-write suppression) and the new directory index. Merging them is
  possible later but not required.
- New Wails bindings and events follow ADR 0013.

## Open questions

- Should the 16-root cache budget or two-worker concurrency limit become
  configurable after measuring real workspace sizes?
- Should the `MAX_FILES` entry cap become configurable? Truncation is now shown by
  the file-list status icon.
- Should `$GLOBAL` / `$NOTES` eventually display nested files, beyond the current
  immediate-child policy?

## Reference

- `frontend.go` - `GetNotesContext`, `SyncNotesContext`, `ListFilesForContext`
- `utils/notes/walk.go` - context resolution, captured-root scan, `MAX_FILES`
- `utils/notes/file_index.go` - cached root snapshots and scan lifecycle
- `frontend_notes_index.go` - snapshot bridge and startup prewarming
- `tmux/window_tab.go` - captured known workspace directories
- `utils/notes/pwd.go` - `DirProjectRoot` and the `~/Documents` fallback
- `utils/file/watcher/watcher.go` - existing single-document watcher
- `utils/file/dirwatch` - macOS/Linux directory watches and polling recovery
- `utils/notes/file_index_watch.go` - batched reconciliation and delta publication
- `utils/notes/file_changes.go` - known-write notifications
- `virtualterm/term.go` - `SetFocus` → `RefreshNotes`
- `virtualterm/osc_functions.go` - OSC 7 → `RefreshNotes`
- `window/backend/renderer_webkit/interface.go` - `RefreshWindowList` → `RefreshNotes`
- `window/backend/renderer_webkit/notes.go` - deduplicated context publication
- `frontend/src/notes.js` - `notesContextChanged` handler, guarded `refreshFiles`
- ADR 0010 - Wails shared state synchronisation
- ADR 0013 - Wails binding checklist
- ADR 0023 - Single-flight async initialisation

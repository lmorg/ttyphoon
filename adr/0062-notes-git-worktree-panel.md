# 0062 - Notes Git panel is scoped to the current project root

## Status

Accepted and implemented on 2026-10-09.

## Context

Notes needs Git status and file-level actions without making Git command
execution or repository parsing part of the frontend. The active Notes project
root may be a normal repository root, a linked worktree root, or a non-Git
`project.ttyphoon` root. Git remains a visible Tools tab even when operations
are unavailable, so it can explain the current context's limitations.

## Decision

- Put Git command execution and status/diff parsing in `utils/git`; expose thin
  Wails wrappers from `frontend.go`.
- Cache Git status in Go by canonical project root. Reuse the cached snapshot on
  reads and explicitly refresh after stage/unstage or a file-index watcher
  completion; publish refreshed snapshots to the frontend. Keep a frontend cache
  keyed by project root so switching Tools tabs or worktrees can rehydrate
  immediately.
- While the Git Tools tab is selected, watch the per-worktree Git metadata
  directory and the shared Git common directory resolved by `git rev-parse`.
  Debounce metadata changes before refreshing status; stop both watches when the
  Git tab is deselected. Watch metadata writes as well as entry creation/removal,
  but exclude object, hook, and reflog trees. Ordinary working-tree file changes
  continue to refresh status through the Notes file-index watcher.
- Run panel status reads with `git --no-optional-locks status` so Git does not
  persist optional index refreshes and retrigger its own metadata watcher.
  Stage/unstage and worktree mutations retain their normal locking behavior.
- Updated 2026-10-22: always show the Tools Git tab, preserving its selection
  across project changes even when Git is unavailable. Show the unavailable
  reason in the panel, including missing project roots and backend Git/status
  failures. Hide the entire Git toolbar and commit controls while unavailable
  or checking a new root, then restore the toolbar after a successful status
  read. Only enable operations when the project root is a usable Git root and
  the host has Git. Accept both `.git` directories and worktree `.git` files.
  The Worktree menu lists all worktrees
  registered with the current repository, sorted by path, and marks the active
  worktree. Selecting one switches Notes' project context while preserving the
  active terminal pane and its other Notes context roots.
- Place a `Branches` button after `Commits`. Its menu starts with Create new
  branch and a separator, followed by sorted local branches. Creation uses the
  shared text prompt and creates/checks out the branch. A branch submenu offers
  copy name, checkout, Fetch latest, and Create new worktree. Checkout reports
  `Switched to <branch>` on success and Git errors on failure without forcing.
  Fetch targets the configured upstream (or same-named branch on origin), never
  checks out or pulls. Branch fetch runs through the interactive Git action
  policy and `ClonePane`, so credential prompts are available in the retained
  terminal pane. Success is determined by the command exit status and reports
  one `Fetch completed for <branch>` notification; failure reports one error
  notification using captured terminal output, falling back to the command
  error. ClonePane inherits the active tile's terminal environment.
  Worktree
  creation uses the default directory plus branch name and the shared creation/
  switch flow. Git commands remain in Go behind thin Wails bindings.
- Place a `Tags` button after `Commits`, with icon `0xf02c` and tooltip
  `manage tags`. Offer tag creation, pull-tags, and local tags ordered by
  descending semantic version. A tag submenu provides copy, push-only-that-tag,
  detached checkout, a detached worktree at the default directory plus tag
  name, and local deletion. Pull-tags and push-tag use the interactive Git
  action policy and `ClonePane`, allowing credentials to be entered in the
  retained terminal pane. Successful commands report one completion
  notification; failures report one error notification using captured terminal
  output, falling back to the command error. ClonePane inherits the active
  tile's terminal environment.
- Git toolbar controls use Font Awesome icons: Commits `0xf0ca` with tooltip
  `View commit log`, Branches `0xf126` with tooltip `Manage branches`, and
  Worktrees `0xf802` retaining its existing tooltip. Keep accessible labels and
  the Commits toggle's pressed state independent of the visible glyph.
- Diff-file loading/opening and stage/unstage failures use the application
  notification system at error severity, including branch-comparison diffs and
  Stage all. Keep the Git file list intact and restore failed action controls
  instead of replacing the panel with an inline operation error. Repository
  status/unavailability messages retain their existing panel presentation.
- Provide a worktree creation modal with a branch picker listing local branches
  alphabetically and an option to create a branch. Default the destination to
  `$HOME/Documents/ttyphoon/.git_worktrees/<sha256>/<branch>`, hashing the
  remote origin URL or, when no origin is configured, the canonical project
  root. Accept relative paths within the project and absolute destinations;
  validate branch names, reject symlinked destination parents, and invoke Git
  with argument-separated commands. Add an in-repository destination to the
  shared repository `info/exclude` so a nested worktree does not appear
  untracked.
- Start creation from Notes asynchronously. Close the modal immediately, create
  a sticky `Creating worktree` terminal notification before invoking Git, and
  invoke the Worktrees menu's Switch action after successful creation, using the
  returned canonical path. Both dialog and commit-menu creation use this shared
  flow; close the notification after switching completes. Report failures as
  terminal error notifications.
- Removing a worktree removes only its working directory/registration and never
  deletes its branch. Do not force removal; let Git reject dirty worktrees.
- Render branch state, ahead/behind counts, and changed paths in the Git pane.
  The `Commits` toggle is the first right-aligned Git toolbar item, before
  Worktrees. It replaces the left sidebar file tree
  with topologically ordered history, while Tools continues to show Git status.
  Keep the shared file/grep filter visible during history mode. Use the same
  `FilterStrings` query language against full commit hashes and complete messages,
  including message bodies and tags, while still displaying short hashes and subjects.
  A non-empty history filter hides the graph and uses the full sidebar width for
  commit text. Clearing it restores the unfiltered graph and text positioning.
  Search a cached window of up to 10,000 commits; repeated queries reuse it,
  with request guards rejecting stale filter results. Restore file/grep filtering
  on history exit; keep the file tree mounted and updated in the background.
  Updated 2026-10-26: show a default-off `Search filenames` checkbox below the
  shared filter only while history is displayed. When enabled, apply the same
  query language to individual changed repository-relative paths and include a
  commit when any path matches; do not also match hashes, messages, or tags in
  this mode. An empty query still shows all commits. Fetch changed paths only
  for a nonempty filename query, using a single NUL-delimited raw Git log batch
  that includes root commits and separate merge diffs. Disable rename detection
  so both old and new paths are searchable, include deletions, and deduplicate
  paths repeated across merge parents. Cache file-aware snapshots separately
  from metadata-only results; reject stale responses after query/mode changes.
  Hide the checkbox in file mode and reset it on project-root changes. Normal
  metadata filtering does not load changed paths.
  History includes only commits reachable from the checked-out worktree's HEAD,
  including merged ancestry, not unrelated branches, remote refs, or tags.
  Detached HEAD uses its own ancestry; an unborn branch shows no commits.
  Go reads Git's graph layout and
  NUL-delimited short hash, first-line subject, author date/time, and author name;
  display the parenthesized author on a separate line below the date/time.
  the frontend draws the graph with the project tree's one-pixel lines in theme
  yellow. Only the portion behind the text is masked to 0.7 alpha; hovering text
  dims its graph to 0.2 opacity, while hovering exposed graph dims its text by
  the same factor. A row resize observer keeps the overlap mask aligned and is
  disconnected when history is replaced, hidden, or its project changes. The renderer
  connects lanes using adjacent graph lines rather than fixed slash offsets.
  Adjacent commit SVGs share boundary coordinates, so expanding/collapsing lanes
  remain continuous even when commit text makes rows different heights. Merge
  dots connect to subsequent lanes instead of being drawn as horizontal dashes.
  Opposite-direction slashes in the same column retain that lane. Leftward
  collapse connectors are kept even where a vertical continuation also exists;
  tests verify lane coverage, not just matching endpoints between commit rows.
  Same-direction diagonal continuations take precedence over adjacent vertical
  rails. Route diagonals first so crossing rails cannot create a second, false
  input to a slash that already continues from another slash.
  The graph is anchored to the left while commit text is anchored to the right. Wide graphs
  may overlap text; text is painted above the graph and retains up to 70% of the
  list width (or its natural width, if smaller), using more space when available.
  Commit rows use the grep-result hover style. Commit subjects use the Markdown paragraph
  font (Lato) at the Notes minus-two
  font size. Load 100 commits initially, with explicit
  Load more batches up to 10,000. Turning the toggle off restores files. Workspace
  changes reset the toggle, including workspaces sharing the same project root.
  A left-side comparison selector defaults to `HEAD`; local branches are listed
  alphabetically with the checked-out branch omitted. Selecting another branch
  changes only the diff base, never the checked-out branch. In that mode the
  file list contains paths differing from the selected branch plus untracked
  working-tree files, and each file diff uses that branch as its original side.
  Clicking a history commit (or pressing Enter/Space on its row) selects that
  commit through the same comparison handler as the branch menu. The selector
  shows its short hash; comparison APIs use its validated full commit ID.
  This refreshes the file list and diff base without checking out the commit.
  Right-clicking a history row opens a menu titled with its short hash: Copy
  commit hash, separator, Compare commit, Create worktree. Copy uses the full
  hash; Compare uses icon `0xe13a` and invokes the left-click handler. Create worktree targets the
  commit in detached mode at the default worktree directory plus its short hash,
  while Git still receives the full hash. Detached worktree menu labels use
  Git's short abbreviation of their HEAD instead of `detached HEAD`,
  reusing the dialog's shared notification-and-creation handler and Wails binding.
  Show a provider-brand button before Refresh only for GitHub, Azure DevOps,
  GitLab, or Bitbucket origins. Resolve HTTPS browser URLs in Go using the
  origin and current branch; open them through the system browser. Hide the
  button when origin is missing, unsupported, or the worktree is detached.
  Selecting a path opens a side-by-side Monaco diff comparing `HEAD` with the
  working tree in a dedicated top-level Notes `Diff` tab. The modified side is
  editable for regular text files; binary and symbolic-link diffs are read-only.
- Open the selected path through Notes' normal document loader. The diff's
  modified model stays synchronized with the regular Notes editor model, so
  filename display, recent documents, history, autosave, and save hooks remain
  shared with ordinary files.
- Keep separate Monaco editor and diff-editor factories because Monaco exposes
  different widget APIs, but share keyboard/completion, clipboard, and context
  menu hooks between them.
- Support file-level stage and unstage actions. Save dirty diff edits before
  staging or unstaging the same file. Select operations use argument-separated
  Git invocations and paths relative to the validated root.
- Place a Commit toggle with icon `0xf386` after the comparison selector on the
  left side of the toolbar. The existing commit-history toggle moves to icon
  `0xf022`. The Commit toggle shows/hides a row directly below the toolbar.
  The row contains a themed single-line `Commit message` input and a Commit
  button on the left, with Stage all aligned right to the existing 72-by-24
  Staged/Unstaged actions. The row is hidden initially; changing project roots
  hides it and clears any unsent message.
- Updated 2026-10-27: a file-filter row normally occupies the second toolbar
  line. Showing commit controls replaces that row; hiding them restores it
  without clearing the query. Use the shared `FilterStrings` syntax against
  displayed paths in both working-tree and revision-comparison lists. Preserve
  file order and actions while hiding nonmatches; show `No matching files` for
  an empty result. Escape clears the query. Reapply it after status refreshes,
  reset it on root changes, and reject stale asynchronous results. Hide both
  secondary rows when Git operations are unavailable. The filter uses the same
  input styling and native-text-assistance disablement as other Git inputs,
  without enabling dictionary spellchecking.
- Stage all runs `git add -u -- .`: stage modifications and deletions to tracked
  files, but do not stage untracked files. Disable the button when workspace
  status has no unstaged tracked files. Refresh Git status after successful
  staging.
- The Git command result runner logs successful stdout and stderr lines at info
  level, and does not info-log output from failed commands. Commit reports one
  `Commit created` notification on success, or one error notification using
  stderr with the command error as fallback.
- Commit requires a non-empty trimmed message and invokes `git commit -m` with
  the message as one argument. It commits the current index only; it does not
  stage changes implicitly. A successful commit clears the input and refreshes
  status. The empty staged-index case is reported as a Git error.
- Show Git path and status-code hover help through Notes' shared tooltip rather
  than native webview tooltips. Keep tooltip behavior delegated from the Git
  pane so dynamically rendered status rows use the same path.
- Handle filenames as NUL-delimited Git status records, including spaces and
  rename source paths. Cap text diff inputs and reject path traversal and
  symlink escapes.

## Consequences

- The panel follows the active project context. A root change refreshes status;
  a non-Git root hides the tab.
- Branch comparison is view-only: `HEAD` remains the default status view, while
  selecting a local branch changes the file list and diff base without checking
  out or mutating any branch.
- Updated 2026-10-27: revision comparisons also list files present in the selected
  commit tree that match the working tree. Keep the existing differing-file
  groups and `≠` status first, then append equal files with `=` and a matching
  revision tooltip. Read the selected tree using NUL-delimited `git ls-tree`;
  do not infer equality from a missing working-tree status entry alone. Equal
  entries retain any real staging flags but do not acquire artificial staging
  actions, and remain openable through the normal comparison diff path. Returning
  to `HEAD` restores the ordinary changed-only working-tree list.
- Git metadata status refreshes are watched only while the Git tab is selected.
  Linked worktrees watch their external per-worktree and common Git directories;
  ordinary file edits are covered by the active project file-index watch.
- The top-level Notes Diff tab is available when the active file appears in the
  cached Git status, even when that file was opened from the Notes file list.
  Watcher-published status updates can show or hide the tab as the file becomes
  dirty or clean.
- Selecting a Git file makes it the active Notes document. The normal Notes
  filename, recent list, and history apply; the diff overlays the regular editor
  while keeping its model synchronized. Leaving the diff reveals the regular
  editor for the same file. The Diff tab is present only while a Git diff is
  active. Cmd/Ctrl+S keeps the normal format-and-save behavior.
- If Monaco is disabled, opening a Git file still loads it into Notes and
  updates history/recents, but the side-by-side diff is unavailable.
- Git operations include read, stage, unstage, worktree listing, local branch
  listing, worktree creation, and switching Notes to a registered worktree.
  Branch deletion/renaming, multi-repository discovery, and persistent workspace
  lists are out of scope.

## Reference

- `utils/git/git.go` - repository validation, status, diff, stage/unstage, and
  worktree/branch operations
- `frontend.go` - Wails wrappers
- `window/backend/renderer_webkit/notes.go` - Notes project-context worktree switch
- `frontend/src/notes-git-panel.js` - status panel and file actions
- `frontend/src/shared_tooltip.js` - shared tooltip rendering and pointer tracking
- `frontend/src/monaco_adapter.js` - shared editor hooks and diff-editor facade
- `frontend/src/notes.js`, `frontend/src/notes.css` - tab, workspace, and editor
  integration
- `utils/git/git_test.go`, `frontend/src/notes-git-panel.test.js` - behavior
  coverage
- ADR 0013 - Wails binding checklist
- ADR 0018 - Monaco provider architecture

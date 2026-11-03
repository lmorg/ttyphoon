# 74. Notes history is scoped to workspace names

Date: 2026-10-21

## Status

Accepted.

## Context

Project-root history made independent tmux windows share navigation when they
used the same root, and split a window's navigation when its panes used different
roots. The frontend also compared only the last restored filename and required
history files to be present in the current file index. Identical `$PROJECT` paths
could therefore retain the previous workspace's document, while files outside
the active root could not be restored.

## Decision

- A Notes workspace is a tmux window **name**, not its ID or active project root.
  History uses `notesw_history` keys prefixed with `workspace:`. Windows with the
  same name share history. Renaming a window selects a different history key.
- Persist expanded absolute filenames and saved line positions. A workspace's
  history may contain files from multiple roots or outside every indexed root.
  Returned paths may use Notes aliases when they refer to the current context;
  external files remain absolute. File-list membership is not a restore gate.
- Retain the existing fifty-position Back/Forward stack, including forward-history
  truncation when a new position is added after going Back. Empty stacks return
  navigation boundary errors rather than indexing an empty slice.
- Recent Files is the newest ten distinct filenames from the stored history,
  newest first, including stored forward entries. Both recent-file menus use
  this projection. There is no separate recent-file write path, cache namespace
  initialization, alphabetical sorting, or `MaxRecentFiles` setting.
- Explicit opens, creation, symbol/definition jumps and cursor navigation record
  history. Passive restores and background file reads do not append entries.
- Restore history on workspace-context transitions, including changes to the
  active root within a workspace. Do not suppress restoration based on a global
  filename sentinel. An empty workspace history clears the previous document.
  Cancel pending cursor-history debounce timers during context transitions.
- On first access to a workspace without a workspace-history cache entry, copy
  its active root's legacy history, preserving positions and the selected index.
  Persist even an empty result so later root changes cannot seed it again.
  Existing root histories and the retired recent-file table are left untouched.
  Historical roots cannot all be assigned to window names retrospectively, so
  migration seeds only from the root active on first access.

## Consequences

History now supports navigating multiple project roots within one workspace;
the file index and Git panel remain scoped to their existing active context.
Workspace restoration preserves the cached document tab (ADR 0021). Rename and
delete update the active workspace's history, which automatically updates its
recent-file projection (ADR 0064).

Tests cover migration, workspace isolation, cross-root navigation, empty stacks,
distinct recents, matching aliases across workspaces, and restoration outside
the active file index.
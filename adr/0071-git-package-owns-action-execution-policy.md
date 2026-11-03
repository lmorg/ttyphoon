# 71. Git package owns action execution policy

Date: 2026-10-15

## Status

Accepted. Updated 2026-10-29: branch fetch, tag fetch, and tag push use the
interactive runner so remote credentials can be entered in ClonePane.

## Context

Git actions have two execution paths. Actions such as commit may need an
interactive terminal so hooks, credential prompts, or editors can run; other
actions can execute directly from Go and return captured output. Putting that
choice in `frontend.go` or the Notes JavaScript duplicates Git policy across the
Wails bridge and UI. Making the Git package depend on tmux or the renderer would
couple domain logic to the terminal implementation.

## Decision

- `utils/git` owns the action request model, command construction, and the
  decision whether an action runs through Go or an interactive terminal.
- The Git package executes Go-mode actions itself. For interactive actions it
  calls an injected runner with the complete, shell-quoted command; the Git
  package does not import or depend on tmux or the renderer.
- `commit`, generic `fetch`, configured-upstream branch fetch, tag fetch,
  single-tag push, and branch `push` use the interactive runner. Remote
  operations that may need credentials must be reachable through this path.
  Add or change execution modes only in the Git package.
- `frontend.go` remains a thin Wails adapter: it supplies environment data and an
  interactive runner that forwards the command to the active renderer/tmux pane
  and returns the completion result.
- Frontend JavaScript sends an action name and UI values, then presents the
  returned mode, output, and errors. It does not build Git command-line flags or
  choose the execution path.

## Consequences

Git execution policy has one owner and can be tested without a running tmux
session by injecting a runner. The Wails layer remains responsible only for
bridging an interactive request to the terminal. Adding an action requires
defining its structured request and execution mode in `utils/git`, plus UI
presentation where needed.

## Reference

- `utils/git/action.go` - action translation and execution-mode policy
- `frontend.go` - Wails adapter and interactive runner
- `window/backend/renderer_webkit/interface.go` - active pane cloning
- `frontend/src/notes-git-panel.js` - UI submission and result presentation
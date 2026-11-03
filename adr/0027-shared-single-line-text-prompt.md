# 0027 — Reuse the note modal as a shared single-line text prompt

## Status

Accepted.

## Context

Two unrelated features needed a single line of free-text input from the user:
LSP "rename symbol" and renaming an AI session in the session management
modal. Both were implemented with `window.prompt(...)`, a native browser
dialog that cannot be styled, is blocking, and is inconsistent with every
other input surface in the app.

The app already has a modal purpose-built for exactly this: `#notes-modal`,
used by `openNewFilePrompt()` / `openRenamePrompt()` for creating and renaming
notes. It has a title, a single text input, and Cancel/Create actions, plus an
optional location-selector button that note create/rename turns on. Nothing
about the input/title/actions is note-specific.

## Decision

Generalise `#notes-modal` into a reusable single-line prompt via
`openTextPrompt({ title, value, confirmLabel, placeholder })`, which returns a
`Promise<string|null>` — the trimmed input, or `null` when cancelled (Escape or
Cancel). It hides the location-selector button (irrelevant outside note
create/rename) and restores it on close.

`state.textPromptHandler` holds the pending resolver. `createNewFile()` (the
modal's Enter/Create handler) and `closeNewFilePrompt()` (Escape/Cancel) check
it first and short-circuit to resolving the generic prompt instead of running
note create/rename logic. This means the existing keyboard/Escape/click wiring
for the modal did not need to change — only the two functions that decide what
"confirm" and "cancel" mean.

Both `renameCurrentLspSymbol()` and the AI session rename button now call
`openTextPrompt(...)` instead of `window.prompt(...)`.

Updated 2026-10-14: creating a new AI session from the AI panel's session
dropdown also uses `openTextPrompt(...)`. A supplied name is passed to
`CreateAISession`; confirming an empty value passes an empty summary so the
session database retains its existing `New session` default. Cancelling the
prompt creates no session. The prompt title is `New AI session`, with a `Create`
confirmation action and an optional-name placeholder.

Updated 2026-10-19: shared prompts use the same `markdown-body`/`h3` title
markup as Rename, Delete, and New Worktree, including Git Branch/Tag and AI
Session creation. Dialog titles use the yellow Jura heading style rather than
plain accent-colored prompt text. Shared terminal input-box titles and the AI
settings dialog follow that same heading presentation.

### Folder-targeted file creation

The file-list context menu reuses the same create-file modal for
`Create new file...` and `Paste as new file...`, including storage section
headings such as `$PROJECT` and `$NOTES`. The location selector identifies the
storage root and the filename input is prefilled with the folder-relative path.
Paste reads through `GetClipboardData`, captures the payload before showing
the modal, and suggests `clipboard.md` for text or `clipboard.png` for an image.
Confirmation writes text through `SaveFile` or base64 PNG through
`SaveBinaryFile`; cancellation or a fresh create/rename clears the pending payload.

File right-clicks combine the original Go-provided file actions, including
rename/delete/copy, with the parent-folder Create/Paste actions. They do not
replace the original file actions. Blank-space right-clicks use the nearest
visible folder or section header above the pointer. Folder collapse/expand
controls remain available, falling back to the section's folders when the
target folder has no descendants. Existing descendant-only collapse/expand
behaviour is preserved for folders that have child folders.

## Consequences

- No new DOM/CSS: the prompt looks and behaves like every other modal in the
  app.
- `window.prompt` no longer appears anywhere in the frontend. Any future
  single-line input requirement should use `openTextPrompt`, not reintroduce
  a native dialog.
- The tradeoff is implicit coupling: `createNewFile()` and
  `closeNewFilePrompt()` now have a dual responsibility (note modal vs. generic
  prompt) gated on `state.textPromptHandler`. This is acceptable because both
  functions are already the single entry/exit point for the modal, and the
  branch is a one-line early return.

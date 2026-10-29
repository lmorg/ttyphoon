# 75. Inputbox drafts are separate from submitted history

Date: 2026-10-21

## Status

Accepted.

## Context

Users need to close an unfinished inputbox without executing its action or
losing its main input, additional variables, or pasted images. Submitted history
is not a suitable draft store because a draft is not a submitted action and may
contain typed values such as false, zero, empty strings, and list selections.

## Decision

- Ordinary inputboxes offer Save Draft. Confirmation dialogs never do.
- Updated 2026-10-22: the action row is OK, Draft, History, Cancel. Draft uses
  the icon `0xf13a` and History uses `0xf1da`, with accessible labels and shared
  application tooltips rather than native `title` tooltips.
  OK and Cancel retain their text and behavior.
- Save Draft sends an explicit `saveDraft` result with `isOk=false`. The backend
  retains the dialog's draft key with its callback and invokes Cancel, not OK,
  after storing the draft. No action or submission-history entry is created.
- Store drafts in `inputboxw_draft` for 30 days. Identity is the SHA-256 hash of
  the inputbox options, including initial text, variable definitions/defaults,
  and an explicit CacheKey where provided. Exclude History because adding a
  submitted history item must not make an existing draft inaccessible. An
  explicit `DraftKey` overrides option-derived identity and is itself SHA-256
  hashed, allowing related dialogs to share one stable draft without sharing
  submitted history.
- AI prompt dialogs use one explicit draft key per tmux workspace name, so
  toolbar, skill, URI, and document prompts restore the same latest draft even
  when their titles or variable definitions differ.
- Reopening an identical configuration restores the main input and typed
  variable values, overriding initial defaults even for false, zero and empty
  strings. Legacy comma-separated list options remain intact while restoring
  the selected option. Pasted images are retained for image-enabled dialogs and
  remain subject to the validation and bounds in ADR 0029.
- Save Draft waits for the Wails call before closing and prevents duplicate
  submissions. A rejected bridge call leaves the dialog and edited fields open.
- Cancel, Escape and backdrop dismissal preserve their existing behavior and
  neither overwrite nor delete a stored draft. An accepted OK submission clears
  the stored draft. Invalid image submissions do not clear it.
- Clear a draft by writing a typed nil pointer, which serializes as JSON null.
  An untyped nil is ignored by the cache and must not be used for this purpose.
- Expanded variable history remains a separate feature. The history popup and
  per-field history behavior are unchanged by draft support. Implemented
  separately on 2026-10-21 in ADR 0076.

## Verification

Backend tests cover draft identity, typed cache round-trips, image retention,
history separation, cancellation and clearing. Frontend tests cover restoration,
false/zero/empty values, lists, save acknowledgement/failure and confirmation
exclusion. Browser checks cover save/reopen behavior and narrow-screen layout.
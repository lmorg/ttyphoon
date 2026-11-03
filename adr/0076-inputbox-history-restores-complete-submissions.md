# 76. Inputbox history restores complete submissions

Date: 2026-10-21

## Status

Accepted.

## Context

Inputboxes previously cached only the main text. Selecting a history item could
therefore combine an old prompt with unrelated current variable values. Text
variables also lacked the main input's history shortcut.

## Decision

- Keep the existing `inputboxw_history` namespace, caller/cache key and 365-day
  lifetime, but persist entries containing `value` and typed `variables`.
  Decode legacy string entries as main-text-only records. The caller-facing
  `InputBoxWTOptions.History` remains `[]string` and supplies seed entries.
- Persist history only on accepted OK submission, never on Draft or Cancel.
  Exclude inline image bytes and the draft action flag. Drafts keep their
  independent complete-payload storage from ADR 0075.
- Image-bearing submissions retain validated file references and content
  digests. Full-history selection restores those images lazily through
  `TerminalInputBoxHistoryImages`, identified by the open dialog ID and history
  index, not an arbitrary client-provided file path. Check image opt-in and
  snapshot bounds, restrict files to the application attachment directory,
  revalidate size/format and verify the saved content digest. Cache and initial
  history events contain no base64 image payloads.
- Promote duplicate complete submissions to the top. Identical text with
  different variable sets or image contents remains independently selectable.
  JSON-equivalent numbers and identical image content deduplicate after cache
  round-trips, regardless of the saved image filenames.
- Emit structured `historyEntries` plus the compatible text-only `history`
  projection. New frontends prefer structured entries and accept old payloads.
- Clicking the history button or Ctrl+H in the main input restores the main
  text and every recorded variable through typed setters. Preserve false,
  zero, empty strings and list selections. Reset variables missing from a
  structured variable map to current defaults; unavailable list values fall
  back to the first current option. Legacy text-only records leave variables
  unchanged because their prior values are unknown.
- Full-history labels include variable summaries so repeated prompt text can
  be distinguished, plus attachment counts where present. An empty prompt
  remains selectable. Full selection replaces the current attachment list;
  selecting a text-only entry clears it. Disable submission while selected
  images load and show an error if their saved files are unavailable. Ignore
  late results after another selection or dialog close. Per-field history does
  not alter attachments. Earlier entries that omitted image references cannot
  recover their attachments retrospectively.
- Up Arrow in an empty main text input or string variable opens newest-first,
  distinct values for that field only. Selecting a value does not change the
  other fields. Number controls, checkboxes and list controls keep their
  existing keyboard behavior.

## Verification

Backend tests cover legacy migration, typed values, complete-submission
deduplication, independent variable sets, file-backed image restoration and
omission of inline image bytes/draft data.
Frontend tests cover full restoration, field-local selection, duplicate prompt
text, empty values, attachment replacement/loading/failures and compatibility
with old payloads. Browser checks use the
real popup-menu implementation for complete and field-only keyboard recall.
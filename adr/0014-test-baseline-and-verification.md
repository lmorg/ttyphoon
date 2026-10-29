# 14. Known test baseline and verification commands

Date: 2026-09-06

## Status

Accepted. Updated 2026-10-23: the full frontend suite passes; preserve earlier
verification records and historical Go failures separately.

## Context

`go test ./...` did not pass cleanly, and one failure *panicked*, aborting the
test binary and hiding every result that would have followed it in that package.
Without a recorded baseline each session re-diagnosed the same failures and risked
attributing them to work in progress.

## Decision

Record the baseline and the verification commands, and keep this record current as
failures are fixed.

### Latest verification

On 2026-10-27, all 616 frontend tests across 23 files passed after adding the
Git file-filter toolbar and single-equals comparison status. The full Go suite
and both application/frontend builds also passed. The preceding unchanged-file
comparison change passed all 613 frontend tests before the toolbar tests were
added.

On 2026-10-26, all 612 frontend tests across 23 files passed after adding Git
history filename filtering. The full Go suite, application build, and frontend
production build also passed. Existing chunk-size and duplicate `-lobjc` linker
notices remain warnings, not failures.

On 2026-10-24, the full frontend suite passed all 609 tests across 23 files after
the native text-assistance and single-line spellchecker changes. The production
frontend build also succeeded; its existing chunk-size notices remain warnings.

On 2026-10-23, `npm run test:run -- --reporter=dot` passed all 605 tests across
23 files. `npm run build` succeeded and `npm ls --depth=0` reported a valid
dependency tree. The bundle-size notices are warnings, not build failures.

The investigation initially reproduced seven failures in `src/notes.test.js`.
Fixes were test-only: close the Markdown modal after its print test so its
capture-phase Escape listener cannot interfere with later tests; inspect CSS
declarations and normalize child-combinator whitespace instead of matching
single-line formatting; await XML fixture persistence; and scope the print
test's no-save assertion to the printed document rather than deferred saves
of other documents. Production code and user stylesheet changes were preserved.

### Previous Verification (2026-10-16)

On 2026-10-16, `npm run test:run -- --reporter=dot` ran 508 tests across 20 files:
506 passed and two exceeded Vitest's 5000 ms timeout:

- `src/notes.test.js`: `renders grouped note categories and nested files from the Wails file list`.
- `src/terminal.test.js`: `clears backend hover when the pointer leaves the terminal for Notes`.

These were timeouts, not failed UI assertions. Their cause has not been
established; do not treat this run as a complete pass. The previous full
frontend run passed all 506 tests before the two completion-background cases
were added. The terminal file subsequently passed all 16 tests in isolation.

Previously completed checks passed for `./utils/git`, `./utils/jupyter`,
`./tmux`, `./virtualterm`, `./window/backend/renderer_webkit`, and the root
package. `go build ./...` also passed. Focused tmux race checks cover layout-event
coalescing, non-echoing resize updates, and stale pane metadata after window
closure. The Go recheck was skipped during this documentation update; these are
prior results, not a fresh full-module verification.

The full `utils/git` package passed with `init.defaultBranch=master` supplied
through process-local `GIT_CONFIG_*` variables. Its former worktree-table
failure is no longer part of the failing baseline. The combined branch/worktree
test now uses a strict synthetic Git executable and fixtures: unsorted records,
whitespace-containing paths, exact worktree-creation arguments, and shared
exclude-file entries. It cannot invoke installed Git because its `PATH` contains
only the stub. Other Git integration fixtures explicitly initialize `main`,
rather than inheriting the machine's default branch.

### Historical Go failures

Observed on macOS with Go 1.27.1 on 2026-10-14; these AI packages were not rerun
during this documentation update:

- `ai/tools/file`: `TestInsertLinesCallDescriptionRendersEscapedInserts` and
  `TestPatchFileCallDescriptionRendersEmbeddedTemplate` expect older description
  formatting/line numbers.
- `ai/tools/shell`: `TestBashReturnsOutputAndExitError` expects a returned Bash
  exit error, but receives nil.

At that time, these three failures and the now-cleared Git worktree-table
failure also reproduced in an isolated clean `HEAD` export, without the Notes
navigation/storage changes.

On 2026-10-15, two stale toolbar-placement assertions in `src/notes.test.js`
were updated to match the current UI: commit history sits beside the comparison
selector after a separator, and the commit-controls toggle is the first
actions-toolbar item. Icon, visibility, and action checks remain; no application
UI was changed during that test review.

### Resolved

Both fixed at root cause rather than by adjusting assertions:

1. **`ai/agent` panic** — `findService` called `panic()` for an unknown service.
   Every caller already handled a `nil` return, and
   `SetServiceModelFromSelection` even contained unreachable `if service == nil`
   error handling. The panic contradicted the design of its own callers, and made
   any `Agent` constructed without loaded config unusable. `findService` now
   returns `nil`; `SwitchServiceModel` gained the nil/bounds guard it lacked.

2. **`ai/tools/subagents`** — `TestSubagentToolContracts` asserted
   `strings.Contains(desc, "written summary")`, but the embedded description is
   hard-wrapped markdown containing `"written\nsummary"`. The assertion could
   never match. Comparison now collapses whitespace via `strings.Fields`, so
   reflowing a description no longer breaks the test.

### Commands

```bash
go build ./...
go test ./... -count=1                 # whole module
go test ./ai/... -count=1              # AI subsystem
go test ./ai/agent -count=1 -race      # concurrency-sensitive packages
go test ./ai/agent -run '^$'           # compile/type-check only, runs nothing

cd frontend
npm test -- --run src/notes.test.js --reporter=dot
node --check src/notes.js                            # syntax only
```

Use `npm test` rather than a bare `npx vitest`, which may prompt to install a
different major version.

## Consequences

- A passing focused check does not establish a full-suite pass. Keep historical
  failures, fresh assertion failures, and timeouts distinct when assessing a
  regression.
- **Lesson: prefer fixing the code over relaxing the assertion.** Both failures
  were real defects — a panic that broke testability and contradicted its
  callers, and an assertion that could never pass — not stale expectations.
- **Lesson: a panic in a shared lookup is hostile to testing.** Returning `nil`
  and letting callers decide is both more testable and more robust at runtime.
- `-race` remains worthwhile for the emitter, agent tool maps, and `sessiondb`;
  those concurrency bugs do not reproduce reliably without it.

## Reference

- `ai/agent/services.go` — `findService`, `SwitchServiceModel`
- `ai/tools/subagents/subagent_test.go` — `containsPhrase`
- `utils/runewidth/runewidth_test.go` — passes in the 2026-10-14 verification

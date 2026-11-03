# 68. Proposal: separate Git command output streams

Date: 2026-10-14

## Status

Proposed. Deferred for future consideration; no implementation is implied by
this record.

## Context

The Notes Git panel's fetch and tag-transfer commands use a dedicated runner
that captures stdout, stderr, and process failure separately. Git commonly
writes successful progress and ref-update summaries to stderr. Transfer
reporting now classifies severity by command outcome: successful stderr is
written to the application log at info level, and only a failed command creates
an error notification.

The other Notes Git operations share `utils/git.runGit`. It uses
`exec.Cmd.CombinedOutput`, returning stdout and stderr interleaved as one byte
slice. Most callers parse that output for status, refs, diffs, or history. This
means a successful stderr diagnostic can contaminate machine-readable output,
and the original stream distinction is unavailable to callers.

## Proposal

When revisited, consider changing the shared Git command runner to capture stdout
and stderr independently:

- Return stdout to existing parsers on success, preserving their current data
  contracts.
- On success, send nonempty stderr to the application log at info level, one
  line per log entry. Do not log routine successful stdout from status, diff,
  history, or metadata queries; those are command results, not diagnostics, and
  logging them would flood the Log panel.
- On failure, preserve stderr for the returned error and use the process error
  when stderr is empty. Preserve timeout handling.
- Audit operations whose successful stdout is useful to people (for example
  branch/worktree mutations) separately. Do not discard or automatically surface
  that output as part of the parser-runner change.
- Keep transfer-specific user notifications outcome-based as recorded in ADR
  0062. Do not turn every successful Git command into a notification.

## Open questions

- Should failed stdout be included in the error when stderr is nonempty, or only
  stderr plus the process error?
- Should successful stderr be logged centrally by `runGit`, or should the
  runner return a structured result and let each caller decide what constitutes
  useful diagnostics? Central logging is simpler; structured results preserve
  caller control and avoid logging commands whose stderr is intentionally used
  for expected control flow.
- Which successful mutating commands should surface human-readable output, and
  through which surface: the Notes Git panel, the application log, or both?
- How should output be bounded or truncated for commands that can produce large
  diagnostics?

## Consequences if adopted

Parsed output would no longer mix successful stderr diagnostics into stdout.
Users could inspect successful Git diagnostics in the Log panel without seeing
spurious error notifications. The change must preserve existing exit-status,
timeout, parser, mutation, and notification behavior, and needs tests for both
streams on successful and failed commands.

## References

- `utils/git/git.go` - `runGit` and `runGitWithAgentEnvironment`
- ADR 0062 - Notes Git panel and outcome-based transfer notifications
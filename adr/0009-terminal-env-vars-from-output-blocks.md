# 9. Terminal environment variables come from the latest output block

Date: 2026-09-06

## Status

Accepted

## Context

`Term.GetEnvVars()` backs the `envVars` and `commandLine` tools, the latter using
it to resolve `$SHELL`.

It originally returned `term._blockMeta.EnvVars` — the *current* block's snapshot.
That is unreliable, because `endOutputBlock` immediately allocates a fresh
`BlockMeta` for the next command. Once a command finished, `_blockMeta` pointed at
an empty placeholder and the tools saw nothing.

A Go process cannot inspect the live environment of an already-running child
shell; `os.Environ()` returns ttyphoon's own environment, not the shell's mutated
one. Nor can the environment be inferred from command *output*, which is
arbitrary text.

The shell integration must therefore report it explicitly. The environment
arrives on the `\033_begin;output-block;…` APC sequence and is stored on the
block by `beginOutputBlock`.

Note the bundled `integrations/shell.bash` and `integrations/shell.zsh` send only
the command line, so `EnvVars` is empty for those shells. A shell that does send
the environment (e.g. murex) populates it.

## Decision

`GetEnvVars()` scans **rows** for the most recent block that actually carries an
environment snapshot, newest first:

1. the normal screen buffer, in reverse
2. then the scrollback buffer, in reverse

Rows retain their `*BlockMeta` pointer after the block ends, so the completed
block's data remains reachable even once `_blockMeta` has been replaced.

The method takes `term._mutex` and returns a **copy**, so callers cannot mutate
stored terminal metadata or race with terminal updates. An empty map is returned
when no block has reported an environment.

## Consequences

- The tools see the most recent environment the shell actually reported.
- Correctness depends on shell integration; shells that do not send the
  environment yield an empty map rather than stale or wrong values.
- The reverse scan is O(rows) worst case when nothing has been reported. Fine at
  current buffer sizes; cache on the `Term` if it ever shows up in profiling.

## Reference

- `virtualterm/term.go` — `GetEnvVars`, `cloneEnvVars`
- `virtualterm/apc_functions.go` — `beginOutputBlock`, `endOutputBlock`
- `types/term_row.go` — `BlockMeta.EnvVars`

# 77. AI filesystem tools use the live project root

Date: 2026-10-21

## Status

Accepted.

## Context

Agents retained the project root detected at creation. Changing the owning
pane's working directory therefore left relative filesystem operations pinned
to the original project, even after the pane had moved into another one.

## Decision

- Resolve `Agent.ProjectRoot()` through the owning terminal's tile and current
  working directory each time it is requested. Reuse `notes.DirProjectRoot` for
  Git/project-marker discovery and existing fallback behavior. Remove the
  creation-time root field rather than keeping a second, potentially stale root.
- Do not use the renderer's focused tile. Changing tabs or focusing another
  pane must not redirect a running agent's filesystem operations.
- Bare filenames and relative paths for read/write/patch/insert resolve against
  the live project root through the shared path resolver. Directory listing,
  grep and image-tool path resolution use the same accessor. Existing absolute
  path handling, symlink checks, sandbox restrictions and permission policies
  remain unchanged.
- Tag grep page caches with the root used for that search. A page-only request
  after a root change returns an error asking for a new search instead of serving
  results from the previous project.
- Read-file notifications display the resolved filename. Never slice it using
  the prompt's captured `Meta.Pwd`, which can refer to another, longer directory.

## Verification

Tests change the owning pane's directory between two projects while another
pane is focused, reuse read/write/directory tools after root changes, retain
sandbox rejection of the old root, and reject stale grep pages until a new
search is performed.
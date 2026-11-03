# 0051 - Native tools can declare structured inputs

Date: 2026-09-29

## Status

Accepted. The base tool interface contains lifecycle and metadata only. All
built-in native tools use structured invocation; MCP uses a separate
JSON-schema/string-input capability.

## Context

Native tools exposed a single required string field named `input`, even when
their actual arguments were objects. The model then had to serialize an object
inside that string inside the outer tool-call JSON. This duplicated encoding
and obscured the tool's real fields from the model's schema validator. ADR 0030
made the string compatibility path tolerant of plain JSON objects, but did not
remove the wrapper schema.

## Decision

- Keep `aitypes.Tool` focused on lifecycle and metadata. It does not require an
  invocation method, so a tool can compose only the capabilities it supports.
- Add `aitypes.StructuredTool`: it exposes its Go input type (object, array, or
  scalar) and accepts the decoded typed value through `CallStructured`.
- Isolate the compatibility `Call(context.Context, string)` method in
  `aitypes.StringInputTool`. New native tools should use `StructuredTool`
  instead of implementing this legacy adapter.
- Add `aitypes.JSONSchemaTool` as a composition of `StringInputTool` and a
  declared JSON Schema. MCP tools use this capability to retain their server-
  supplied schema and JSON argument adapter.
- Eino accepts native `StructuredTool` implementations and `JSONSchemaTool`
  adapters. It dispatches native calls through `CallStructured` and invokes
  `Call` only through the explicit string-input capability.
- Generate the Eino JSON Schema from that input type. The model sees the real
  object properties rather than a `{ "input": "..." }` wrapper.
- Decode `argumentsInJSON` once into the declared struct before invoking the
  structured tool. A structured observation provider receives the same decoded
  value, avoiding a second input unmarshal for continuation records.
- Keep MCP tools on their declared JSON Schema/object path and pass their JSON
  arguments to the MCP adapter through `JSONSchemaTool` / `StringInputTool`.
- The native migration is complete: dateTime, readDirectory, grep, readFiles,
  writeFile, insertLines, patchFile, createGraph, generateImage, jq, bash,
  envVars, delegate, report, askUser, searchDuckDuckGo, and
  scrapeWebPage all implement `StructuredTool`.
- `bash` accepts a `{script, summary}` object and executes Bash in a cloned terminal pane
  for both commands and scripts. It replaces the separate `interactive` tool
  (ADR 0056). `envVars` uses an empty JSON object.
- `delegate` and `report` use arrays of `{name, prompt}` objects; `askUser` uses
  a `{question, choices}` object.
- `searchDuckDuckGo` uses a `{query}` object for search and an
  `{action:"discover"}` object for SearXNG discovery, with optional filters and
  refresh behavior described in ADR 0073; `scrapeWebPage` uses a `{url}` object.
- `dateTime` accepts an optional positive amount, unit, and before/after
  direction; omitting them returns the current local date and time.
- Updated 2026-10-21: `readFiles` accepts a `files` array of objects containing
  `path` and optional `startLine`/`endLine` bounds for that individual file.
  One call can mix different ranges and whole-file reads. Legacy string entries
  are accepted as whole-file reads; top-level bounds are explicitly rejected
  rather than silently ignored. Lines are 1-based and inclusive. Omitting both
  bounds for a file reads that whole file; an omitted start means line 1
  and an omitted end means EOF. Reject nonpositive or reversed bounds. Requests
  beyond EOF return available content or empty content if the start is beyond
  EOF. Ranged reads stream through the needed prefix, preserve line endings and
  long lines, honor cancellation, and stop after the selected end rather than
  loading the entire file. Approval descriptions and structured observations
  associate each requested range with its file; the output remains a
  filename-keyed txtar archive.
- jq accepts a structured JSON object with a query and arbitrary JSON value. The
  value is encoded to jq's stdin without an XML wrapper or lossy number decoding.
- Add the optional `aitypes.ToolCallDescriber` contract for tools that need to
  explain a specific invocation before user approval. It receives the same
  decoded input as `CallStructured` or the string input passed to a
  `StringInputTool`, and returns Markdown inserted before the approval links.
  Tools without the contract retain the original approval prompt.

## Consequences

- Structured native tools receive validated, typed fields on their first call.
- Tool descriptions and schemas can agree on nested options and field names.
- All built-in native tools use the structured schema and invocation path; the
  base `Tool` interface no longer forces them to implement `Call`.
- Built-in native concrete types no longer implement the legacy `Call` method.
  The `StringInputTool` capability remains for MCP's JSON-schema adapter.
- Approval descriptions are optional and scoped to the invocation; they do not
  change tool schemas or the behavior of tools without a description method.

## Reference

- `ai/agent/aitypes/aitypes.go` - composable tool and invocation contracts
- `ai/agent/runtime_eino.go` - schema generation and single decode
- `ai/agent/agent.go` - permission request formatting and approval flow
- `ai/tools/file/grep.go` - prototype structured native tool
- `ai/agent/runtime_eino_test.go` - schema and typed invocation regression
- `ai/tools/file/observation_test.go` - grep structured invocation coverage
- `ai/tools/image/generate.go` - structured generation/edit input
- `ai/tools/websearch/tool.go` - structured web search and discovery inputs
- `ai/tools/webtools/web_tools.go` - structured page scraping inputs
- `ai/tools/shell/bash.go` - structured shell input and cloned-pane execution
- `utils/markdownutil` - escaping helpers used by invocation descriptions
- ADR 0030 - legacy native string-input compatibility
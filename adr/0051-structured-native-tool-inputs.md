# 0051 - Native tools can declare structured inputs

Date: 2026-09-29

## Status

Accepted. Grep was the prototype; all native tools in `ai/tools/file`, jq, the
image generator, shell tools, subagents, askUser, web tools, and dateTime now
use the structured contract. Other native tools remain on the legacy
string-input contract until migrated individually.

## Context

Native tools exposed a single required string field named `input`, even when
their actual arguments were objects. The model then had to serialize an object
inside that string inside the outer tool-call JSON. This duplicated encoding
and obscured the tool's real fields from the model's schema validator. ADR 0030
made the string compatibility path tolerant of plain JSON objects, but did not
remove the wrapper schema.

## Decision

- Keep `aitypes.Tool.Call(context.Context, string)` for unmigrated native tools.
- Add the optional `aitypes.StructuredTool` contract: it exposes its Go input
  type (object, array, or scalar) and accepts the decoded typed value through
  `CallStructured`.
- Generate the Eino JSON Schema from that input type. The model sees the real
  object properties rather than a `{ "input": "..." }` wrapper.
- Decode `argumentsInJSON` once into the declared struct before invoking the
  structured tool. A structured observation provider receives the same decoded
  value, avoiding a second input unmarshal for continuation records.
- Keep MCP tools on their declared JSON Schema/object path, and keep existing
  native string tools on the ADR 0030 compatibility path.
- Migrate native tools case by case. Grep is the prototype; readFiles,
  readDirectory, writeFile, insertLines, patchFile, jq, and generateImage now
  use the same path.
- `commandLine` uses a direct JSON string; `envVars` uses an empty JSON object.
- `delegate` and `report` use arrays of `{name, prompt}` objects; `askUser` uses
  a `{question, choices}` object.
- `searchDuckDuckGo` uses a JSON string; `scrapeWebPage` uses a `{url}` object.
- `dateTime` accepts an optional positive amount, unit, and before/after
  direction; omitting them returns the current local date and time.
- jq accepts a structured JSON object with a query and arbitrary JSON value. The
  value is encoded to jq's stdin without an XML wrapper or lossy number decoding.

## Consequences

- Structured native tools receive validated, typed fields on their first call.
- Tool descriptions and schemas can agree on nested options and field names.
- The migration is incremental; only tools implementing `StructuredTool` use
  the new schema and invocation path.
- The legacy string contract remains available but is no longer required for
  new native tools.

## Reference

- `ai/agent/aitypes/aitypes.go` - optional structured-tool contracts
- `ai/agent/runtime_eino.go` - schema generation and single decode
- `ai/tools/file/grep.go` - prototype structured native tool
- `ai/agent/runtime_eino_test.go` - schema and typed invocation regression
- `ai/tools/file/observation_test.go` - grep structured invocation coverage
- `ai/tools/image/generate.go` - structured generation/edit input
- `ai/tools/web_tools.go` - structured web search and page scraping inputs
- ADR 0030 - legacy native string-input compatibility
# 0030 - Native tools accept plain JSON arguments

## Status

Accepted.

## Context

Native ttyphoon tools are exposed to the model through a single-string
compatibility schema:

```json
{"input":"<raw tool input>"}
```

The model occasionally emits the natural tool object directly instead:

```json
{"prompt":"...","size":"1024x1024","quality":"high"}
```

The runtime previously passed that plain object unchanged to native tools. JSON
native tools then attempted to unmarshal it as the raw input contract and
returned errors such as `input must be valid JSON ... unexpected end of JSON
input`.

MCP tools are different: their schemas are real object schemas and their
arguments must remain objects when sent through the MCP client.

## Decision

Normalize only the native-tool compatibility boundary in
`einoAgentTool.InvokableRun`:

- `{"input":"..."}` unwraps to the string, as before;
- `{"input":{...}}` unwraps and JSON-encodes the nested value;
- a plain JSON object or array is JSON-encoded and passed as the native tool's
  raw input string;
- a JSON string remains the raw string;
- malformed input remains unchanged so the tool can return its normal recoverable
  error.

MCP tools bypass this normalizer and continue receiving the object produced from
their declared schema.

## Consequences

- First-attempt calls to JSON native tools work whether the model emits the
  compatibility wrapper or the natural object shape.
- Existing raw-string native tools remain compatible.
- MCP schema semantics are not altered or flattened.
- The tool-call progress display continues to show the model's original
  arguments, while the delegate receives the normalized form.

## References

- `ai/agent/runtime_eino.go` - `unwrapToolInput`
- `ai/agent/runtime_eino_test.go` - wrapped and plain JSON cases
- `ai/agent/mcp_tool.go` - MCP object argument path

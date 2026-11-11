# Approved on-demand MCP startup

## Status

Accepted.

## Context

Agents need to discover useful configured MCP servers without starting every
server at the beginning of a task. Starting a process or connecting to a service
requires an explicit, understandable approval by default.

## Decision

- Define `startMcpServer` in `ai/tools/mcp`, with its description embedded from
  Markdown. The agent owns catalog validation and MCP lifecycle operations;
  the tool uses a narrow capability interface rather than the concrete agent.
  Register it in `ai/tools` so the tool package depends only on shared types,
  not the agent implementation. This also lets agent integration tests exercise
  the real tool without an import cycle.
- Enable it by default with `askPermission` invocation policy. Reuse existing
  per-invocation/per-prompt permission handling. Approval details include the
  server name and a required model-provided justification.
- Never delegate startup to non-interactive sub-agents.
- Accept only an exact `filename::server` identifier discovered through
  ttyphoon's configured MCP files; omit the config filename's `.json`
  extension. Do not accept new commands, URLs, or arbitrary configuration paths.
- Add an optional server `description`. Advertise only the identifier and
  description to the model, not transport settings or credentials.
  Treat descriptions as data, not instructions. Invalid configuration files
  produce an explicit catalog error rather than a silently incomplete catalog.
- Reuse existing input collection, OAuth, loading reservations, atomic tool
  registration, and notifications. Derive startup cancellation from the calling
  task's context.
- Eino binds tool schemas when a ReAct graph is created. Return directly after a
  startup tool batch, retain all generated model/tool messages, and bind a fresh
  graph before continuing the same task. Do not replay completed tools.
  Denials and failures remain model-visible and recoverable; cancellation ends
  the task. Bound repeated startup continuations.

## Consequences

Existing configurations remain valid without descriptions. Starting a server
does not grant additional permission to invoke its tools; their existing defaults
and AI settings overrides remain in force.

Tests cover default approval, sub-agent prohibition, safe catalog metadata,
configured-only validation, approval wording and denial, parent cancellation,
and same-run use of newly registered tools, including failed/denied continuation.

## Reference

- [Tool permission scopes](0002-tool-permission-scopes.md)
- [Interactive links](0007-ttyphoon-uri-interaction-links.md)
- [Structured native tools](0051-structured-native-tool-inputs.md)
- [MCP cancellation and retry](0078-mcp-oauth-failures-must-be-retryable.md)
- [MCP configuration options](../ai/mcp_config/options.md)

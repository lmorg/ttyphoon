# Agent Runtime Boundary

This note defines the runtime boundary used during the Eino migration.

## Ownership

- Agent runtime orchestration
  - Owned by ai/agent package.
  - Entrypoint: Agent.RunLLMWithStream.
  - Runtime selection: newPreferredRuntime (Eino only).

- Runtime implementation
  - Owned by ai/agent/runtime_eino.go.
  - Responsibilities:
    - initialize provider-specific chat models (OpenAI, Anthropic, Ollama)
    - build and run ReAct agent
    - stream model output chunks to UI callback
    - map MCP tools into Eino tool config

- Tool execution contract
  - Owned by ai/agent/tools.go plus concrete tool implementations.
  - Runtime treats tools through agent.Tool abstraction and does not depend on provider-specific tool SDK types.

- Streaming contract
  - Runtime emits incremental text chunks via stream callback.
  - MCP tool progress is emitted as Action/Action Input chunks to preserve existing frontend pipeline formatting.

- Error handling
  - Runtime returns native runtime/provider errors.
  - No legacy string-based parse fallback behavior.

## Phase Gate Intent

- Phase 0 complete when this boundary note exists and matches implementation.
- Phase 1 complete when agent runtime entrypoints are runtime-boundary based with no legacy SDK runtime implementation in ai/agent.

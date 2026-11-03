# 78. MCP OAuth failures must be retryable

Date: 2026-10-22

## Status

Accepted.

## Context

Cancelled authorization left loopback callback listeners waiting independently
of the setup context. Retrying could find the callback port occupied and ask
the user to paste a URL. Concurrent loads could both pass an existence check,
and registration published a server before all its tools were prepared. These
paths could leave partial or duplicate MCP state.

The Go SDK v1.8.0 additionally validates the OAuth authorization response issuer
per RFC 9207. Notion's public authorization-server metadata advertises this
requirement, but the callback adapter previously discarded the `iss` value.
The frontend hid the resulting reason behind a generic load-failure message.

## Decision

- OAuth callbacks are automatic only. A listener failure is an actionable
  connection error, never a request to paste a callback URL. Keep the original
  exported hook fields compatible, but do not invoke manual callback hooks.
- Callback waits honor setup cancellation and the two-minute callback limit.
  Close the loopback listener on every exit, including cancellation, timeout,
  provider denial and failed exchange. Reject stale callback states without
  consuming the active attempt, preserve `iss`, and let the SDK validate it.
- Thread a setup context through HTTP/stdio connection and tool discovery.
  Setup has a three-minute bound and the loading notification's Cancel action
  cancels it. Normalize cancelled setup errors against the context even when
  the SDK reports a closed session. Successful sessions outlive setup-context
  cancellation. Close failed, superseded and unused clients.
- Serialize authorization within a handler. Initial authorization failure is
  terminal for that connection's handshake; a new load gets a fresh handler.
  A concurrent stale 401 can retry the newly obtained token without another
  browser flow. A 401 rejecting a still-valid freshly authorized token reports
  an error instead of reopening the browser. Preserve SDK step-up handling for
  insufficient-scope responses.
- Reserve each server name per agent before setup. Reject concurrent loads and
  conflicting configuration sources. Release reservations on all exits.
- Prepare and validate every advertised tool before committing the client,
  source and full tool set under lifecycle/tool locks. Reject duplicate tool
  names and duplicate server commits. Failed setup changes neither server nor
  tool registration. Reload the agent runtime only after committing, outside
  the lifecycle lock.
- Unload removes server/tool state atomically before closing the client. Agent
  shutdown cancels pending loads and prevents late/new registration. Do not
  hold registry locks across connection close, OAuth UI or renderer callbacks.
- Replace token files atomically using private-permission temporary files, so
  concurrent refreshes cannot expose partially written JSON.
- Disable pending MCP UI toggles, refresh authoritative state after failures,
  and include the backend error in the application error notification. Retrying
  does not require restarting the AI session.

## Verification

Local tests use the real SDK client/server transports and a mock authorization
server. They cover occupied callback ports without manual fallback, cancelled
OAuth followed by successful automatic retry, required issuer validation, stale
state, provider denial, rejection without a login loop, session survival after
setup, atomic/concurrent registration, failed-load rollback, and concurrent
token persistence. Targeted lifecycle tests also run under the race detector.

Notion's public metadata was checked without account access. Live authenticated
Notion sign-in remains a manual verification step.
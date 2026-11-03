# 0085 - Migrate the desktop application to Wails v3

Date: 2026-11-02

## Status

Accepted

## Context

Wails v3 replaces the v2 application/runtime model. The project had one large
`WApp` binding facade, Wails runtime calls spread through backend packages, and
frontend imports tied directly to v2-generated paths. Migration also affects
startup, events, window close behavior, asset serving, generated bindings, and
the platform build workflow.

The v2-to-v3 migration page shows a JSON `wails.json` example, but the pinned
v3.0.0 CLI uses `build/config.yml` and a Taskfile-based build workflow.

## Decision

- Keep `WApp` as one v3 service for this migration. Do not combine the runtime
upgrade with a domain-service reorganization.
- Keep core logic in backend packages. Expose Wails operations through the
Wails-independent `types.DesktopRuntime` interface and the context adapter in
`utils/wailsruntime`; do not pass `*application.App` through backend code.
- Preserve event payloads for existing frontend handlers through
`frontend/src/wails-runtime.js`. Emit a frontend-ready event to replace the v2
DOM-ready callback.
- Use `build/config.yml` and `Taskfile.yml` for Wails v3 builds, development, and
binding generation. Keep `@wailsio/runtime` pinned to 3.0.0 to match the Go
module.
- Keep `ttyphoon://` links handled inside the app as required by ADR 0007, but
do not register the scheme with the operating system.
- Use the standard v3 About dialog with the app name, description, and icon;
the custom v2 macOS About content is not carried forward.
- Preserve maximized startup, hidden macOS title bar, hide-on-close behavior,
the notes asset handler, and the product identifier `com.lmorg.ttyphoon`.

## Consequences

- v3 service and runtime types stay at the app adapter boundary; existing
backend synchronization decisions in ADR 0010 remain unchanged.
- Generated bindings are recreated with `wails generate bindings` and imported
through the stable local wrapper described by ADR 0013.
- The Wails v3 CLI (`wails`) must be installed and available on `PATH` for Makefile and
Taskfile commands.
- The migration guide's JSON config example is not used with the pinned v3.0.0
CLI; revisit if the project later upgrades to a CLI version that supports it.
- External application launches through `ttyphoon://` are not registered;
in-app interactive links remain supported.

## References

- [Wails v2-to-v3 migration guide](https://wails.io/migration/v2-to-v3/)
- [ADR 0007: interactive `ttyphoon://` links](0007-ttyphoon-uri-interaction-links.md)
- [ADR 0010: Wails shared-state synchronization](0010-wails-shared-state-synchronisation.md)
- [ADR 0013: Wails binding checklist](0013-wails-binding-checklist.md)
- `frontend.go`, `wails_runtime.go`, `utils/wailsruntime/runtime.go`
- `build/config.yml`, `Taskfile.yml`, `frontend/src/wails-runtime.js`
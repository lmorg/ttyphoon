# Architecture Decision Records

Decisions and hard-won knowledge for ttyphoon, recorded so future work can build
on them rather than rediscover them.

Each record uses: **Status**, **Context**, **Decision**, **Consequences**,
**Reference**.

`0000-high-level-project-context.md` provides a high level architectural description of the project and should be read to provide context required for all other ADRs.

## Index

| # | Title | Area |
|---|---|---|
| [0001](0001-tool-failures-must-not-abort-the-agent-run.md) | Tool failures must not abort the agent run | AI runtime |
| [0002](0002-tool-permission-scopes.md) | Tool permission scopes are per-invocation or per-prompt | AI permissions |
| [0003](0003-max-step-continuation.md) | Resume the agent after max-step exhaustion | AI runtime |
| [0004](0004-ai-request-timeout-configuration.md) | AI request timeout is global configuration | Config |
| [0005](0005-ai-panel-emit-gating.md) | AI panel emits are gated on what is actually on screen | AI panel |
| [0006](0006-parallel-subagent-output-buffering.md) | Parallel sub-agent output is buffered, not streamed | Sub-agents |
| [0007](0007-ttyphoon-uri-interaction-links.md) | Interactive `ttyphoon://` links for agent-to-user round trips | UI protocol |
| [0008](0008-ask-user-tool.md) | `askUser` tool for blocking clarification | AI tools |
| [0009](0009-terminal-env-vars-from-output-blocks.md) | Terminal environment variables come from the latest output block | Terminal |
| [0010](0010-wails-shared-state-synchronisation.md) | Wails-facing shared state must be explicitly synchronised | Concurrency |
| [0011](0011-execution-limits-visibility.md) | Report harness limits; do not invent provider limits | AI settings |
| [0012](0012-system-prompt-conventions.md) | System prompts state their context and constraints | Prompts |
| [0013](0013-wails-binding-checklist.md) | Adding a Wails binding requires three coordinated edits | Frontend |
| [0014](0014-test-baseline-and-verification.md) | Known test baseline and verification commands | Testing |
| [0015](0015-pane-maximize-background-stretch.md) | Pane maximize stretches the background pane | Layout |
| [0016](0016-notes-lsp-integration-as-built.md) | Notes LSP integration as built | LSP |
| [0017](0017-notes-lsp-gaps-and-doc-drift.md) | Notes LSP gaps and documentation drift | LSP |
| [0018](0018-lsp-monaco-provider-rendering.md) | Semantic tokens and inlay hints render through Monaco providers | LSP |
| [0019](0019-lsp-references-find-mode.md) | LSP references use a dedicated Find result mode | LSP |
| [0020](0020-sleep-wake-hang-hardening.md) | Sleep/wake hang hardening | Concurrency |
| [0021](0021-editor-navigation-must-target-the-live-surface.md) | Editor navigation must target the live editor surface | LSP |
| [0022](0022-workspace-symbols-server-side-search.md) | Workspace symbols must be searched server-side | LSP |
| [0023](0023-single-flight-async-initialisation.md) | Single-flight async initialisation | Concurrency |
| [0024](0024-dismiss-menu-before-dispatching-selection.md) | Dismiss a menu before dispatching its selection | UI |
| [0025](0025-subagent-tool-robustness.md) | Sub-agent tool robustness and unique notification IDs | AI |
| [0026](0026-subagent-tool-delegation.md) | Sub-agent tool delegation | AI |
| [0027](0027-shared-single-line-text-prompt.md) | Reuse the note modal as a shared single-line text prompt | Frontend |
| [0028](0028-go-backed-notes-table-sorting.md) | Reuse the Go table engine for Notes table sorting and filtering | Tables |
| [0029](0029-ai-image-attachments-render-in-the-stream.md) | AI image attachments render as Markdown images in the stream | AI panel |
| [0030](0030-native-tool-arguments-accept-plain-json.md) | Native tools accept plain JSON arguments | AI runtime |
| [0031](0031-ai-settings-history-stays-bounded.md) | AI Settings history stays bounded regardless of what produced it | AI panel |
| [0032](0032-image-tool-sandboxing-and-provider-shapes.md) | Image tool sandboxing, and OpenRouter's different edit shape | AI tools |
| [0033](0033-ai-settings-ordering-is-not-uniform.md) | AI Settings ordering is deliberately not uniform | AI panel |
| [0034](0034-pin-to-viewport-toolbars-avoid-sticky.md) | Pin-to-viewport toolbars use scroll-synced transform, not `sticky` | Frontend |
| [0035](0035-ai-output-explicit-follow-mode.md) | AI output uses explicit follow mode | AI panel |
| [0036](0036-multiple-notes-surfaces-with-shared-workspace-services.md) | (design reverted) Multiple Notes surfaces with shared workspace services | Notes |
| [0037](0037-ai-stream-is-addressable-blocks-in-sqlite.md) | The AI stream is a set of addressable blocks persisted in sqlite | AI panel |
| [0038](0038-typed-stream-blocks-must-honour-panel-scope.md) | Typed stream blocks must honour workspace and prompt scope | AI panel |
| [0039](0039-remove-legacy-markdown-compatibility-layer.md) | Scheduled removal of the legacy markdown compatibility layer | AI panel |
| [0040](0040-transient-model-stream-disconnects-use-checkpoint-retry.md) | Transient model stream disconnects use checkpoint retry | AI runtime |
| [0041](0041-rerun-failed-ai-prompts-from-persisted-checkpoints.md) | Rerun failed AI prompts from persisted checkpoints | AI runtime |
| [0042](0042-large-ai-stream-footprint-and-ui-responsiveness.md) | Large AI streams must preserve UI responsiveness | AI panel |
| [0043](0043-adjusted-terminal-cells-centre-glyphs.md) | Adjusted terminal cells centre glyphs | Terminal |

## Recurring themes

**Distinguish recoverable from terminal.** A tool that fails, is denied, or is
disabled should inform the model and let the run continue. Only cancellation and
timeout should end it. (0001, 0002)

**Persistence and presentation are separate concerns.** The transcript is always
written; emitting to the panel is conditional on what the user is actually
looking at. (0005, 0037)

**Mutex-safe is not the same as correctly serialised.** Garbled output usually
means several logical writers share one sink, not a data race. (0006, 0010, 0037)

**A new transport must inherit the old transport's gating.** Adding a second
emit path beside an existing one, without routing it through the same
suppression predicate, reintroduces every bug the predicate existed to prevent.
(0038)

**Date-stamp compatibility layers when you add them.** A shim kept for one
reason spreads across every layer it touches, and none of those layers records
why it is still there. (0039)

**Retry transport failures through state, never by replaying side effects.** A
provider stream can fail after tools have already run; retry the checkpoint,
not the original request. (0040)

**Rerun failures as new linked requests, never as tool replay.** Preserve the
failed audit trail and let the model verify state before taking side effects.
(0041)

**Bound the UI by frames, not tokens.** Large streams must coalesce rendering
and transport work without dropping durable output. (0042)

**Give each writer its own addressable sink.** Serialising writes onto one
append-only buffer orders the bytes but not the logical blocks; the fix is to
stop sharing the sink, not to add another lock. (0037)

**Prefer `Unknown` to a plausible guess.** Displayed values are used to reason
about failures; a wrong one is worse than an absent one. (0011)

**Scope is explicit.** Invocation, prompt, run, and session are different
lifetimes and must not be conflated. (0002, 0003, 0005)

**Symmetrical-looking features are often asymmetric underneath.** Copy-pasted
implementations drift, and identical code can still behave differently when the
context it runs in is not identical. (0015)

**"OpenAI-compatible" is a spectrum, not a boolean.** Two providers claiming
the same API can implement genuinely different request shapes for the same
feature; branch by provider explicitly rather than papering over it with a
generic retry. (0032)

**The same query can have two correct orders.** A shared data-loading function
can have callers with genuinely different, both-correct requirements (context
reconstruction vs. display); don't assume one caller's order is "the" order.
(0033)

**CSS features that look equivalent aren't, in a WebView.** `position: sticky`
and a scroll-synced `transform` can look identical when static, but only one
of them was actually free of repaint side effects in this renderer; prefer
the compositor-only property when a container has translucent backgrounds.
(0034)

**Automatic scrolling must yield to user intent.** A live output surface may
follow the newest content by default, but user scrolling pauses that behavior
until an explicit "Latest" action resumes it. (0035)

**Duplicate views do not imply duplicate ownership.** Document-sensitive state
belongs to each Notes surface, while workspace services such as AI sessions and
global logs retain one authoritative owner and are referenced by focused
surfaces. (0036)

**Verify docs against code before relying on them.** Planning documents describe
intent at a point in time; they drift in both directions, claiming features that
were never built and denying ones that since were. (0016, 0017)

**Search state belongs to the invocation, not the selected result.** Opening a
file from a search result must not silently change what the result list means.
(0019)

**A hidden fallback surface makes "wrong target" bugs silent.** Successfully
manipulating an invisible element raises no error, so the feature looks broken
rather than misdirected. Compare against the sibling code path that works. (0021)

**Probe the server before blaming its configuration.** Identical symptoms — a
feature that renders nothing — traced to a missing gopls setting in one case and
a malformed client request in another. Measure, don't infer. (0018, 0022)

**A truthy guard before an `await` does not serialise anything.** Memoise the
promise, not the result — and never let `void asyncFn()` swallow the failure.
(0023)

**An `await` can be load-bearing.** Code that is only correct because every
caller happens to suspend first will break when an unrelated call is removed.
Order operations explicitly rather than relying on timing. (0024)

**Wall-clock timestamps are not identifiers.** Any loop creating more than one
item per millisecond will collide — and code that matches on the "unique" value
then corrupts unrelated state. (0025)

**An unimplemented interface assertion fails silently and forever.** `x.(iface)`
returning false is indistinguishable from "feature disabled". Put a
`var _ Iface = (*T)(nil)` guard at the assertion site. (0026)

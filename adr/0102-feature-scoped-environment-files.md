# 0102 - Feature-scoped environment files

## Status

Accepted and implemented, 2026-11-16.

## Context

Before this change, application startup called `config.ReadEnvConfig`. It
discovered every `*.env` under the application directory in each
`xdg.ConfigDirs` entry, read them in directory/glob order, and wrote their
contents into the application process environment. A name without a leading
`!` was also added to the list filtered from terminal child environments; `!`
removed that filtering for the name. The loader logged values and did not check
all read/set errors. Files were implicitly shared across features, affected
unrelated process and SDK environment lookups, and could expose secrets in
debug output.

The desired model is explicit opt-in: configuration names the files and the
resulting values belong to the feature that references them. Existing
launch-environment values remain available as fallbacks. This is a deliberate
breaking change for users relying on automatic `*.env` loading.

## Decision

### File lookup and parsing

- Stop the startup `*.env` scan and its `os.Setenv` and dynamic terminal
  exclusion side effects. An unreferenced file has no effect, regardless of its
  name or location. Do not remove or modify users' files.
- Add `EnvFiles []string` to the relevant feature configuration, including
  workspace terminal settings, AI settings/services, and Social/S3. Give the
  MCP server's existing `env` configuration a corresponding `envFiles` field.
  Add feature-local file support for other configuration-backed environment
  reads, including terminal theme expansion and LSP initialization options.
- An absolute entry identifies that file directly. For a relative entry, try
  `<dir>/<app.DirName>/<entry>` for **every** directory in `xdg.ConfigDirs`,
  in the same directory order used by today's discovery. Load every match in
  that order, not just the first. This is not relative to the process working
  directory, the workspace, or the YAML file. Entries are explicit paths, not
  implicit `*.env` globs. A relative name must remain within the application
  config directory; use an absolute path for files elsewhere.
- Process `EnvFiles` entries in list order; within each relative entry, process
  matching directories in the order above. Later files override earlier files.
  Resolve each entry independently, and report a missing entry or an unreadable
  or malformed matching file through the feature's existing error path; do not
  silently use a partial configuration. Missing candidates in individual XDG
  directories are normal when the entry exists in another directory.
- Support the existing literal `NAME=value` format, splitting at the first
  `=`; values can contain `=` and an empty value is still present. Do not
  execute the file as shell code or expand variables in its values. The legacy
  `!` prefix is not supported; reject prefixed names as invalid rather than
  exporting or silently rewriting them. Names use ASCII letters or `_`, followed
  by ASCII letters, digits or `_`. Ignore blank lines and comments; report
  invalid names, malformed non-comment lines, open/read/close errors and invalid
  paths without printing values. Never log resolved values or credentials.

### Precedence and isolation

`utils/envvars.Resolve(envFiles, envs)` loads a feature's scoped values.
`Lookup` falls back to the host environment, while `Overlay` applies scoped
values to an explicitly supplied base without adding other host values. Both
preserve explicit empty values and do not mutate their inputs or the host.

For each environment key, from lowest to highest precedence:

1. The application host environment, or the existing sanitized/inherited
   child environment when constructing a process.
2. Referenced files, in the order described above.
3. Explicit feature `Envs` or `Env` entries. Presence, not non-emptiness,
   decides an override: an explicit empty value must not fall through to a
   file, host environment, or SDK default.

Feature-specific literal configuration fields, such as S3's `AccessKeyID`,
remain authoritative for that field; environment resolution supplies fields
that are not configured. Preserve feature validation after resolution.

Only the selected feature gets the file values. Do not use `os.Setenv`, even
temporarily around SDK initialization: other concurrent features and children
must not observe them. Pass explicit values to SDKs and per-process environment
slices/maps instead. Do not expose them in errors, logs, frontend responses,
AI tool descriptions, transcripts, or unrelated child processes.

- **Workspaces:** `Workspaces.Defaults` and each named terminal gain
  `EnvFiles`. Load default files followed by named-terminal files; merge default
  `Envs`, then named-terminal `Envs`, after all files. Thus any explicit
  `Envs` entry, including a default, wins over any file entry, while named
  entries win over defaults within the same tier. Overlay the result on the
  existing sanitized tmux startup environment and use the exact same effective
  map for tmux `-e` arguments and pane startup snapshots. Quote arguments using
  the shared tmux quoting helper. Use the same scoped lookup for environment
  expansion of the workspace's configured `Pwd`. Existing cloned panes, new
  windows, and splits continue to inherit their source pane's reported
  environment; do not reapply workspace settings to a split.
- **AI:** add AI-wide `EnvFiles` for AI operations that check the host
  environment, and service `EnvFiles` beside `AIServiceT.Env`. For a selected
  service, layer host, AI-wide files, that service's files, then its `Env`.
  AI-wide checks without a selected service use the AI-wide files over host.
  Image generation uses its configured image service's files and `Env`, not
  the chat service's, when an image service is configured; otherwise it uses
  the selected chat service. Main agents and subagents use the same resolver.
  Web search and MCP OAuth checks must also use the appropriate AI-scoped
  resolver rather than bypassing it with `os.Getenv`.
  Search tools refresh their configuration on the next use after a successful
  settings reload; OAuth handlers capture settings when they are created.
  Interactive API-key setters remain explicit host-environment changes, which
  cannot override a service's scoped values.
  The old Ollama model-discovery helper is dormant and remains host-only;
  it must be scoped per service before it is enabled.
- **MCP command servers:** load the server's `envFiles` and overlay its existing
  `env` on the inherited host environment, then pass the resulting environment
  only to that server process. Do not leak chat-service credentials into MCP
  merely because a server was started by an agent. This intentionally fixes
  the present behavior where a nonempty `env` replaces rather than overlays the
  inherited process environment. File contents remain literal; existing input
  substitutions in `env` still apply. Automatic reconnects reuse the server's
  scoped startup settings; restart the server to load changed files.
- **S3:** add `Social.S3.EnvFiles`. A present S3 config may fill missing literal
  credential/region fields from `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`,
  `AWS_SESSION_TOKEN`, and `AWS_REGION` (falling back to
  `AWS_DEFAULT_REGION`); resolve each name from its scoped files over host.
  Explicit S3 fields win. An absent S3 section still disables uploads, and
  incomplete/invalid resolved settings still error. Do not introduce the AWS
  default credential chain, profiles, or instance metadata; bucket, endpoint,
  prefix and public URL remain explicit literal fields.
  `AWS_DEFAULT_REGION` is consulted only when `AWS_REGION` is absent, not
  when it is present and empty.
- **LSP:** configured language servers and `Notes.SpellCheck.TyposLsp` each
  have `EnvFiles`. Resolve their files at process startup and automatic restart,
  overlaying the host environment. Expand `initializationOptions` using the
  same snapshot passed to the process. Running servers retain their existing
  settings; new servers use updated references after a configuration reload.
  File errors use the existing server-start error path, including the optional
  typos-lsp fallback to aspell. Server stderr, log/progress notifications and
  RPC error messages redact scoped values. Like other subprocesses, a language
  server must be trusted with the environment it receives.
- **Terminal theme:** `Terminal.EnvFiles` applies only to expansion of
  `Terminal.ColorTheme`, not to terminal processes. After merging configuration
  files, resolve these files over the host environment and load the expanded
  theme; retain the existing `~` home-directory expansion. Reload rereads the
  referenced files, removed references no longer apply, and a failed reload
  restores the previous theme settings. Theme lookup errors and debug logging
  must not expose scoped values.
- **SDK-owned lookups:** account for environment reads performed inside Go
  libraries, not just ttyphoon's direct `os.Getenv` calls. Explicitly supply
  resolved values through the SDK's supported client options/configuration and
  prevent host or profile fallbacks from defeating an explicitly present empty
  scoped value. Audit all supported AI providers and construction paths,
  including subagents and image generation. If an SDK does not expose a safe
  per-client option, reject unsupported scoped settings explicitly rather than
  mutate the process environment or silently ignore the scoped value. The
  Anthropic adapter rejects scoped profile, federation and webhook settings,
  and rejects scoped empty credentials or headers without an effective nonempty
  credential. Scoped empty provider endpoints are also rejected.
- File resolution errors must reach the operation initiating the feature, and
  configuration reload must use the updated file references without leaking
  prior values into other scopes. Existing live processes need not be mutated.

### Current environment and process inventory

The following inventory records the code before this migration, not current
implementation status.
`Cmd.Env == nil` means Go inherits the application environment. It matters
because the old startup loader changes that environment globally.

| Area | Existing reads, writes or inheritance | Change/impact to evaluate |
|---|---|---|
| Startup/config | `main.go` adjusts macOS `PATH` and calls `config.ReadEnvConfig`; `config/env.go` scans `.env`, uses `os.Setenv`, filters names, and builds terminal exports. `utils/file/config.go` defines XDG order. `config/config.go` loads YAML before `main` and expands `Terminal.ColorTheme` with `os.ExpandEnv`. | Remove only implicit file loading; preserve the fixed terminal sanitization/exports. Resolve theme files before expanding the theme, including on config reload. |
| Workspace/panes | `utils/workspace/workspace.go` merges default and named `Envs`; `tmux/workspace.go` expands `Pwd`, passes `-e`, and seeds pane state. `tmux/tmux.go`, `tmux/window.go`, `tmux/key_functions.go`, `tmux/pane.go`, and `virtualterm/term.go` handle tmux startup, inherited environments and snapshots. `tty/unix/exec_unix.go` uses `config.SetEnv`; Windows has a separate shell path. | Resolve workspace files without changing inheritance or snapshot rules in ADRs 0009 and 0072. Do not turn a workspace file into a global tmux-server environment. Check the non-tmux paths separately. |
| AI providers | `ai/agent/services.go` currently checks nonempty service `Env`, then `os.Getenv`. `ai/agent/runtime_eino.go` and `ai/subagent/subagent.go` use it for OpenAI, Anthropic and Ollama; `ai/tools/image/generate.go` uses the image-service resolver. | Apply layered lookup and presence-aware precedence consistently. The Anthropic SDK also consults `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_BASE_URL`, `ANTHROPIC_MODEL`, headers, profiles and federation-related environment/config; explicitly audit its SDK fallback and options rather than assuming app-level reads are exhaustive. |
| Other AI | `ai/tools/websearch/config.go` directly reads `TTYPHOON_WEB_SEARCH_PROVIDER`, `TTYPHOON_SEARXNG_URL`, `TTYPHOON_SEARXNG_HEADERS` and `TTYPHOON_SEARXNG_MAX_CONCURRENT`. `ai/mcp_client/oauth_auth.go` reads `TTYPHOON_MCP_OAUTH_ENABLE_PRM_SHIM`. `ai/envvars.go` has interactive API-key setters that mutate the host environment. | Route reads through the appropriate scope and review interactive setters for coexistence with explicit scoped values. Preserve web-search validation/redaction and OAuth behavior. |
| MCP and AI subprocesses | `ai/mcp_config/json.go` defines per-server `Env`; `ai/mcp_client/gosdk_adapter.go` passes it as `Cmd.Env` if nonempty. `ai/agent/ollama.go` explicitly inherits `os.Environ`; `ai/tools/jq/cmd.go` inherits implicitly. The Bash tool runs in a cloned pane. | Scope server file values to that server and audit Ollama discovery versus the selected service. Bash retains the pane's environment, not the AI service's secrets. |
| Social/S3 | `config/social.go` defines literal S3 fields; `utils/s3/upload.go` constructs credentials directly and currently does **not** check host environment or use the AWS default chain, despite using the AWS signer. ADR 0096 explicitly records that behavior. | Add the agreed credential/region fallback and update ADR 0096 and configuration documentation when implemented. |
| LSP | `config/config.go` defines `LspT`; `utils/lsp/process.go` expands environment references in initialization options and starts the server with an inherited environment. Both configured language servers and `Notes.SpellCheck.TyposLsp` use this configuration. | Add per-LSP files; expand options and construct its process environment from that server's effective scope, without sharing with other servers. |
| Git | `utils/git/git.go` uses a host-derived environment with `GIT_TERMINAL_PROMPT=0` for one runner; `runGit` otherwise inherits host. Interactive Git actions in `utils/git/action.go` use ClonePane. | Preserve Git-specific prompting and pane inheritance; unrelated feature files must not leak into direct Git commands. |
| Other subprocesses | `utils/jupyter/run.go`, `utils/grep/cmd.go`, `utils/file/meta/type.go`, `utils/menu_hyperlink/open_with.go`, `utils/getshell/{getent,dscl}.go`, `utils/spellcheck/spelling.go`, `window/frontend/browser/user_agent.go` and `ai/tools/jq/cmd.go` launch processes inheriting host environment. `utils/dispatcher/dispatcher.go` explicitly adds its IPC variable to `os.Environ`. | Preserve existing inheritance unless a feature explicitly gains its own file scope; removal of global `.env` loading intentionally removes the accidental file values. |
| Other expansion and mutation | `utils/workspace/pwd.go` and `tmux/key_functions.go` expand workspace paths. `virtualterm/apc_functions.go` can explicitly export/unset host variables; `utils/jupyter/langs.go` and `window/frontend/main/notes.go` use their own template/path mappings rather than general host lookups. | Do not mistake unrelated template variables for host environment reads. Explicit APC host mutations remain host behavior; they do not grant access to another feature's files. |

Third-party SDKs and subprocesses can themselves consult additional host
variables. Their normal host fallback remains available where it does not
contradict an explicit scoped value; enumerate and test provider-specific
fallbacks during implementation rather than assuming this table covers
dependency internals forever.

### Verification and migration

- Test absolute paths, every matching XDG directory, ordering across files and
  directories, missing/unreadable/malformed files, names (including rejection
  of the legacy `!` prefix) and empty values.
  Verify no automatic file loading and no host environment mutation or value
  logging.
- Test file-versus-host and explicit-versus-file precedence, workspace
  defaults/named overrides, tmux arguments and pane seed consistency, and
  isolation between workspaces, services, image services, MCP servers and
  unrelated subprocesses.
- Test provider SDK fallback, including explicit empty values, concurrent
  services with different credentials, web-search authentication/redaction,
  OAuth, S3 credential and region fallbacks, and S3 literal-field precedence.
  Verify reload and error propagation without secret disclosure.
- Document that existing automatically discovered files must be listed under
  each intended feature and that legacy `!` prefixes must be removed from
  variable names. Update the user configuration examples, README, ADRs 0073 and
  0096, and any affected feature documentation when implementation lands.

## Consequences

- Configuration becomes explicit and credentials stop being implicitly shared
  with the application and its children. The same file can be listed by
  multiple features when sharing is intentional.
- Users must migrate existing `.env` configurations; removing auto-loading can
  change theme expansion, providers and child processes that relied on it.
- Correct SDK handling and config reload require more than replacing direct
  `os.Getenv` calls. They must preserve feature isolation under concurrency.

## Reference

- `config/env.go`, `config/config.go`, `config/social.go`, `utils/file/config.go`
- `utils/envvars/envvars.go`, `utils/envvars/envvars_test.go`
- `utils/workspace/workspace.go`, `tmux/workspace.go`, `tmux/pane.go`
- `ai/agent/services.go`, `ai/agent/runtime_eino.go`, `ai/subagent/subagent.go`
- `ai/tools/image/generate.go`, `ai/tools/websearch/config.go`
- `ai/mcp_config/json.go`, `ai/mcp_client/gosdk_adapter.go`
- `utils/lsp/process.go`, `utils/s3/upload.go`
- ADRs 0009, 0072, 0073, and 0096

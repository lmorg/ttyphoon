# Configuration reference

## Files and locations

TTYphoon reads user configuration from `<config-dir>/ttyphoon/`. Here
`<config-dir>` means an entry in `xdg.ConfigDirs` (normally populated from
`XDG_CONFIG_DIRS`, or platform defaults). The separate `XDG_CONFIG_HOME` is
not automatically searched by these readers unless it is also in that list.
On macOS the default list includes `~/Library/Preferences`,
`/Library/Application Support`, `/Library/Preferences`, and `~/.config`.

| File | Purpose | Discovery |
| --- | --- | --- |
| `*.yaml` | Main application settings | All files immediately in each config directory, in directory and filename order |
| `mcp/*.json` | MCP server definitions | Discovered for selection or on-demand startup |
| Explicit environment files, e.g. `secrets.env` | Feature-scoped `NAME=value` settings | Only when listed in `EnvFiles` or `envFiles` |
| `themes/*.itermcolors` | iTerm2 colour themes | Theme menu and `Terminal.ColorTheme` |
| `agent-skills/*/SKILL.md`, `skills/*/SKILL.md` | AI skills | One directory level below either skill directory |
| `AGENTS.md` | Additional AI instructions | Matching file in each config directory |
| `system-prompts/<name>` | Prompt text for terminal AI integration | Loaded when that integration requests the name |

These are runtime user files. The repository's
[`config/defaults.yaml`](../config/defaults.yaml),
[`config/reserved_words.yaml`](../config/reserved_words.yaml),
[`utils/jupyter/languages.yaml`](../utils/jupyter/languages.yaml), and
[`utils/syntaxcompletion/default_rules.yaml`](../utils/syntaxcompletion/default_rules.yaml)
are embedded defaults, not separately discovered user files.
[`build/config.yml`](../build/config.yml) configures the Wails build and
development tooling, not runtime user settings.

## Main YAML files

The application loads embedded defaults, embedded reserved words, then each
user `*.yaml` file. Later files are decoded onto the current configuration;
some map-valued settings have additional merging behavior. YAML keys are
case-sensitive and unknown fields are rejected. Consult
[`defaults.yaml`](../config/defaults.yaml) for current defaults and
[`config.go`](../config/config.go) for the complete Go definitions.

### Terminal, shell and window

| Key | Type | Purpose |
| --- | --- | --- |
| `Tmux.Enabled` | boolean | Enable tmux integration. |
| `Shell.Default`, `Shell.Fallback` | list of strings | Shell command and arguments when tmux is unavailable. |
| `Hotkeys.PrefixTTL`, `Hotkeys.RepeatTTL` | integer | Prefix and chained-key timeouts in milliseconds. |
| `Hotkeys.Functions` | map of string to string | Map a key or `prefix::key` to a function name. |
| `Terminal.ColorTheme` | string | iTerm2 theme path or filename; expands environment references and `~`. |
| `Terminal.EnvFiles` | list of strings | Environment files used only to expand `ColorTheme`. |
| `Terminal.PaneAutoCloseDelay` | string | Go duration for successful interactive-action pane auto-close. |
| `Terminal.ScrollbackHistory` | integer | Scrollback size. |
| `Terminal.ScrollbackCloseKeyPress`, `Terminal.WriteMarkdownHistory` | boolean | Scrollback close behavior and Markdown history output. |
| `Terminal.JumpScrollLineCount` | integer | Terminal line-update interval. |
| `Terminal.AutoHyperlink`, `Terminal.OSC8Hyperlinks` | boolean | Automatic links and application-provided OSC 8 links. |
| `Terminal.Widgets.Table.ScrollMultiplierX`, `.ScrollMultiplierY` | integer | Table scroll multipliers. |
| `Terminal.Widgets.AutoHyperlink.IncLineNumbers` | boolean | Include line numbers in detected file links. |
| `Terminal.Widgets.AutoHyperlink.OpenAgents` | list of objects | Link-opening applications: `Name` (string), `Command` (string list), `Schemes` (string list). |
| `Terminal.Widgets.AutoHyperlink.CustomRegexp` | list of objects | Each entry has `Match` (Go regular expression) and `Link` (replacement string). |
| `TypeFace.FontName` | string | Primary font name. |
| `TypeFace.FontSize`, `TypeFace.AdjustCellWidth`, `TypeFace.AdjustCellHeight` | integer | Font size and cell padding. |
| `TypeFace.Ligatures` | boolean | Enable font ligatures. |
| `Window.StatusBar`, `Window.HoverEffectHighlight`, `Window.AlwaysOnTop` | boolean | Window appearance and behavior. |
| `Window.BellVisualNotification`, `Window.BellPlayAudio` | boolean | Visual and audible bell behavior. |
| `Window.RefreshInterval` | integer | Forced refresh interval in milliseconds; `0` disables the timer. |

Open-agent command arguments can use `${path}` and `${url}` substitutions.
`Terminal.EnvFiles` is read again when settings reload; its values are not
passed to terminal panes.

```yaml
Terminal:
  EnvFiles: [theme.vars]
  ColorTheme: "$THEME_PATH"
```

### Notes, language servers and executable notebooks

| Key | Type | Purpose |
| --- | --- | --- |
| `Notes.MaxFileSizeMB`, `Notes.StructViewMaxSizeKB` | integer | File and structured-view size limits. |
| `Notes.MaxLogLines` | integer | Maximum log lines. |
| `Notes.ExcludeDirectories` | map of string to boolean | Directories excluded from Notes. |
| `Notes.Languages.<language>.TabSpaceIndent` | integer | Indentation width; falls back to `default`, then four. |
| `Notes.Languages.<language>.ReservedWords` | list of strings | Words excluded from spellchecking. |
| `Notes.Languages.<language>.LSP` | object | Language-server settings below. |
| `Notes.Languages.<language>.Jupyter` | list of objects | Executable-notebook language bindings below. |
| `Notes.SpellCheck.TyposLsp` | object | Language-agnostic spellchecker using the same LSP settings. |

Language keys are normalized to lowercase.

#### Language servers (LSP)

Both `Notes.Languages.<language>.LSP` and `Notes.SpellCheck.TyposLsp` support:

| Field | Type | Purpose |
| --- | --- | --- |
| `command` | list of strings | Executable followed by arguments; install the server separately and make it available on `PATH`, or use an absolute executable path. |
| `initializationOptions` | map | Server-specific options sent during initialization; string values can reference environment variables. |
| `EnvFiles` | list of strings | Environment files scoped to this server, overriding its inherited application environment. |

Keys are case-sensitive: `command` and `initializationOptions` start with
lowercase letters, while `EnvFiles` does not. A minimal Go configuration is:

```yaml
Notes:
  Languages:
    go:
      LSP:
        command: [gopls]
```

For server arguments, initialization options and a scoped environment:

```yaml
Notes:
  Languages:
    go:
      TabSpaceIndent: 4
      LSP:
        command: [gopls, serve]
        EnvFiles: [gopls.env]
        initializationOptions:
          buildFlags: ["-tags=integration"]
          env:
            GOPRIVATE: "$GOPRIVATE"
```

The referenced `gopls.env` might contain:

```text
GOPRIVATE=git.example.org
```

Initialization option names and value types depend on the language server;
the options above are for `gopls`. Environment references in nested option
strings use the same snapshot as the server process.

The optional language-agnostic spellchecker has its own configuration:

```yaml
Notes:
  SpellCheck:
    TyposLsp:
      command: [typos-lsp]
      EnvFiles: [typos.env]
```

Each server receives only its own files, which are read when it starts or
restarts. Changing settings does not alter a running server. An empty command
or an unavailable `typos-lsp`, including a failure to load its environment
files, falls back to aspell. Scoped values are redacted from server stderr,
log/progress notifications and RPC errors, but the server process itself
receives those values: only configure trusted language servers.

#### Executable notebooks

A Jupyter binding supports `Aliases` (string list), `PathRegexp` (string),
`Description` (string), `Template` (Go template string), `FileExtension`
(string without a leading dot), and `FormatCommand`, `BuildCommand`,
`BuildCommand2`, `ExecuteCommand`, `ExecuteParameters` (string lists).
Command arguments can use `${FILE}`; parameterized execution also uses
`${PARAMETERS}`. See the [embedded language bindings](../utils/jupyter/languages.yaml)
for examples.

### Tmux workspaces

`Workspaces.Defaults` and each entry in `Workspaces.Terminals` accept the
same fields:

| Field | Type | Purpose |
| --- | --- | --- |
| `Pwd` | string | Initial working directory, with environment expansion. |
| `Cmd` | string | Initial command. |
| `EnvFiles` | list of strings | Scoped environment files. |
| `Envs` | map of string to string | Explicit environment overrides. |

Default files are followed by named-terminal files; explicit default `Envs`
are then followed by named-terminal `Envs`. Thus explicit values override all
files, and named entries override defaults within their tier. These settings
apply to tmux workspace startup; splits and cloned panes inherit their source
pane environment.

```yaml
Workspaces:
  Defaults:
    EnvFiles: [shared.env]
  Terminals:
    project:
      Pwd: "$PROJECT_ROOT"
      EnvFiles: [/secure/project.env]
      Envs:
        PROJECT_ROOT: /path/to/project
```

### AI

| Key | Type | Purpose |
| --- | --- | --- |
| `AI.OAuthUseBuiltinWebview` | boolean | Use the built-in OAuth login window instead of an external browser. |
| `AI.EnvFiles` | list of strings | AI-wide environment files. |
| `AI.MaxIterations`, `AI.MaxContinuations` | integer | Agent iteration and continuation limits. |
| `AI.RequestTimeout` | string | Go duration for AI requests. |
| `AI.DefaultService` | string | Default service key. |
| `AI.ToolSummariseThresholdChars` | integer | Tool-output length threshold for summarization. |
| `AI.Services` | map of service key to object | Provider and model settings below. |

Each service supports `Label`, `Description`, `Provider`, `ImageGenService`,
`DefaultModel`, `SummariseModel` (strings), `Models` (string list),
`EnvFiles` (string list), and `Env` (string-to-string map). Providers include
`openai`, `anthropic`, and `ollama`. If `SummariseModel` is omitted, the
default model is used. `ImageGenService` identifies another service by its
`Label`; image generation uses the selected chat service when it is omitted.

Host values are overridden by AI-wide files, service files, then service
`Env`. Web search and MCP OAuth use AI-wide files, not service files. The
selected service's settings also apply to subagents. API keys entered through
the application menus modify the host environment and do not override scoped
values. MCP command servers have their own `envFiles` and `env` settings.

The Anthropic adapter rejects scoped profile, federation and webhook settings.
Scoped credentials or headers are also rejected when there is no effective
non-empty Anthropic credential, as are empty scoped provider endpoints. These
restrictions prevent SDK defaults from bypassing scoped settings.

```yaml
AI:
  EnvFiles: [ai-shared.env]
  Services:
    private-openai:
      Label: Private OpenAI
      Provider: openai
      Models: [gpt-5.4]
      DefaultModel: gpt-5.4
      EnvFiles: [/secure/openai.env]
      Env:
        OPENAI_BASE_URL: https://example.com/v1
```

For MCP OAuth, `TTYPHOON_MCP_OAUTH_ENABLE_PRM_SHIM` can be set in an AI-wide
file or the host environment to `1`, `true`, `yes`, or `on` to force the
compatibility shim. The shim relaxes issuer validation and normally activates
only for an issuer mismatch.

#### SearXNG web search

Web search defaults to DuckDuckGo. To use SearXNG, add an AI-wide environment
file and reference it from `AI.EnvFiles`:

```yaml
AI:
  EnvFiles: [searxng.env]
```

For example, `searxng.env` can contain:

```text
TTYPHOON_WEB_SEARCH_PROVIDER=searxng
TTYPHOON_SEARXNG_URL=https://search.example.org
TTYPHOON_SEARXNG_HEADERS={"Authorization":"Bearer REPLACE_WITH_TOKEN"}
TTYPHOON_SEARXNG_MAX_CONCURRENT=2
```

The provider and URL are required to select SearXNG; headers and concurrency
are optional. The URL must be an absolute HTTP(S) instance URL without
credentials, query parameters, or fragments; path prefixes are supported.
Headers must be a JSON object of names to values (maximum 32 KiB). Use HTTPS
for authentication except on loopback. Keep real credentials out of
source-controlled configuration. Concurrency defaults to 2 and accepts
integers from 1 to 16. These settings are AI-wide and override the application
environment; service-specific environment files do not configure web search.

The SearXNG instance must enable JSON responses in its own `settings.yml`:

```yaml
search:
  formats:
    - html
    - json
```

See the [web-search guide](AI-Web-Search.md) for discovery, filters, result
handling and rate limits, and
[MCP configuration options](../ai/mcp_config/options.md#command-server-environment)
for command-server environments.

### Social/S3

`Social.S3` is optional; omitting it or setting it to `null` disables clipboard
image uploads. A present but incomplete section is an error.

| Field | Type | Required | Purpose |
| --- | --- | --- | --- |
| `EnvFiles` | list of strings | No | Scoped files for credential and region fallbacks. |
| `Endpoint` | string | No | HTTPS S3-compatible service root, optionally with a path; defaults to `https://s3.<Region>.amazonaws.com`. Custom endpoints must support path-style requests. |
| `Region` | string | Yes | Signing region; falls back to `AWS_REGION`, then `AWS_DEFAULT_REGION` only if `AWS_REGION` is absent. |
| `AccessKeyID` | string | Yes | Signing credential; falls back to `AWS_ACCESS_KEY_ID`. |
| `SecretAccessKey` | string | Yes | Signing secret; falls back to `AWS_SECRET_ACCESS_KEY`. |
| `SessionToken` | string | No | Temporary-credential token; falls back to `AWS_SESSION_TOKEN`. |
| `Bucket` | string | Yes | Bucket name, not a path or URL. |
| `Prefix` | string | No | Object-key directory within the bucket. |
| `PublicBaseURL` | string | Yes | Public bucket/CDN root, excluding `Prefix`. |

Required credentials and region may come from the environment instead of
literal fields. Nonempty literal fields take precedence over scoped files and
host environment fallbacks; literal fields are never environment-expanded.
All listed files must be valid even when literal credentials are supplied.
A present empty `AWS_REGION` does not fall through to `AWS_DEFAULT_REGION`.
AWS profiles, instance metadata and the default credential-provider chain
are not used. Protect credential files and keep them out of source control;
credentials are not sent to the frontend.

```yaml
Social:
  S3:
    EnvFiles: [s3.env]
    Bucket: my-bucket
    Prefix: clipboard/images
    PublicBaseURL: https://cdn.example.com
```

For example, `s3.env` can contain:

```text
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=REPLACE_WITH_ACCESS_KEY
AWS_SECRET_ACCESS_KEY=REPLACE_WITH_SECRET_KEY
```

`Endpoint` and `PublicBaseURL` must use HTTPS without URL credentials, queries
or fragments. Leading/trailing slashes are removed from `Prefix`; `.` and `..`
segments, backslashes, NUL and newline characters are rejected.

#### Publication and clipboard behavior

Configuring S3 opts into uploading local/generated copied images, including
charts and diagrams, without a per-copy confirmation. Existing HTTP(S) image
sources are not uploaded. Each new object has a random UUID filename ending
in `.png`. With the example above, the upload and published URLs are:

```text
https://s3.us-east-1.amazonaws.com/my-bucket/clipboard/images/<uuid>.png
https://cdn.example.com/clipboard/images/<uuid>.png
```

`Bucket` is appended only to the upload endpoint; `Prefix` is appended to both
the bucket path and the public base URL. Public URLs are constructed, not
checked. Configure the bucket/CDN mapping and public-read policy yourself.
Uploads do not set a public ACL or change bucket policy. Credentials need
`s3:PutObject` on the configured prefix, not list or delete permissions.
Random filenames are not access control.

URLs are permanent, not presigned. There is no expiry or automatic deletion;
use bucket lifecycle rules if needed, noting that deletion breaks pasted links.
Identical PNGs within one copy share an upload, but repeated copies create new
objects. Already uploaded objects are not removed if a later upload or
clipboard write fails.

Successful uploads supply public image references for Markdown and Slack
clipboard formats; HTML and RTF retain embedded PNGs. Upload or configuration
failures replace affected images with alt text (or `Image`) and display a
warning while copying remaining content. Conversion and clipboard-write
failures remain errors.

Uploads use a single signed HTTPS PUT with a 30-second timeout, no redirects,
automatic retries or multipart uploads; HTTP 200 is required. PNG limits are
32 MiB encoded, 16384 pixels per side and 16 Mi pixels total. Images upload
sequentially, so several uploads can delay a copy by more than 30 seconds.

## Environment files

Files contain literal `NAME=value` lines. Names start with an ASCII letter or
underscore and continue with ASCII letters, digits or underscores. Blank lines
and `#` comments are ignored; values are not shell-expanded or unquoted.
Empty values are present values. Missing, unreadable or malformed referenced
files cause a feature error without loading a partial result.

Absolute entries are read directly. Relative entries are exact paths, not
globs, searched under `<config-dir>/ttyphoon/` for every `xdg.ConfigDirs`
entry. Every match is loaded in order; paths must stay inside that directory,
including after resolving symlinks. Relative paths are not relative to the
workspace, process working directory or YAML file. Entries are processed in
list order, then matching config directories in directory order.
Later files override earlier files and the host environment.
Explicit `Envs`/`Env` or MCP `env` overrides files,
including with an empty value. Values are scoped to the feature and do not
modify the application host environment.

The supported scopes are `Workspaces.Defaults.EnvFiles`,
`Workspaces.Terminals.<name>.EnvFiles`, `AI.EnvFiles`,
`AI.Services.<name>.EnvFiles`, `Social.S3.EnvFiles`,
`Notes.Languages.<language>.LSP.EnvFiles`, `Notes.SpellCheck.TyposLsp.EnvFiles`,
`Terminal.EnvFiles`, and MCP command-server `envFiles`.

TTYphoon no longer automatically loads `*.env` files at startup. To migrate,
list existing files for each intended feature, or set values in the launch
environment. Remove the old `!` prefix from variable names; prefixed names
are rejected. Files are not deleted or modified. Existing live processes
retain their environments; restart servers or create new workspaces as
appropriate.

### Isolation and reload behavior

Values belong only to the feature referencing the file. For example, AI
service credentials are not passed to an MCP command server or terminal pane.
Child processes inherit their existing base environment plus their own scoped
overrides; explicit empty values suppress lower-precedence values.

- **Workspaces:** new workspace startup uses the configured files and
  overrides. New windows, splits and clones inherit the source pane environment
  instead of reapplying workspace settings.
- **AI:** selected services use AI-wide files, service files, then service
  `Env`. Image generation uses its configured image service, or the selected
  chat service if no image service is set. Subagents use the selected service.
- **Web search and MCP OAuth:** web-search tools pick up reloaded settings
  on their next use; OAuth handlers capture settings when created. Neither
  uses service-specific files. Changing a SearXNG instance's initialized
  concurrency limit requires an application restart.
- **MCP command servers:** automatic reconnects reuse scoped startup settings;
  restart the server to load changed files.
- **LSP:** files and initialization options are resolved together when a server
  starts or restarts. Running servers retain their environment.
- **Terminal themes:** settings reload rereads the files; removed references
  no longer apply. A failed theme reload restores the previous theme settings.
- **S3:** settings reload replaces the S3 configuration; removing its section
  disables uploads.

Missing, malformed or unreadable referenced files produce a feature error,
not a partially loaded environment. Unsupported scoped SDK settings are
rejected rather than silently falling back to host settings. Errors and logs
must not reveal scoped credentials.

## MCP JSON files

Place JSON files under `<config-dir>/ttyphoon/mcp/`. A file can use either
`mcp.servers` as a top-level key or a nested `mcp` object with `servers`
and optional `inputs`:

```json
{
  "mcp": {
    "inputs": [
      {"type": "promptString", "id": "token", "description": "API token"}
    ],
    "servers": {
      "tickets": {
        "type": "command",
        "command": "ticket-mcp",
        "args": ["--token", "${input:token}"],
        "envFiles": ["tickets.env"],
        "env": {"TICKET_PROJECT": "example"}
      }
    }
  }
}
```

| Server field | Type | Purpose |
| --- | --- | --- |
| `description` | string | Description shown in the server catalog. |
| `type` | string | `http` or `https` for HTTP; other values use a command server. |
| `command`, `url` | string | Command executable or HTTP endpoint. |
| `args` | list of strings | Command arguments. |
| `env` | map of string to string | Command-server environment overrides. |
| `envFiles` | list of strings | Command-server environment files; not used by HTTP servers. |
| `oauth` | object | Optional HTTP OAuth settings. |
| `DefaultPermissions` | object | `invocation`: `alwaysAllow` or `askPermission`; `subagents`: `allow` or `deny`. |
| `override` | object | `appName` overrides the MCP client name; `webSite` is parsed but currently unused. |

`mcp.inputs` entries accept `type` (`promptString`), `id`, `description`
and `password` (boolean). `password` is parsed but currently does not change
the input dialog. Input references can be substituted in arguments, environment
values, URLs and supported OAuth fields.

| `oauth` field | Type | Purpose |
| --- | --- | --- |
| `clientId`, `clientSecret`, `clientUri` | string | Client credentials or client metadata document URI. |
| `redirectUri` | string | Local OAuth callback URI. |
| `scopes` | list of strings | Fallback scopes when the server advertises none. |
| `authServerMetadataUrl` | string | Authorization-server metadata URL. |
| `tokenFile` | string | Path for persisted OAuth tokens. |
| `enabled`, `pkceEnabled` | boolean | Deprecated; PKCE is always used. |

See the [MCP configuration reference](../ai/mcp_config/options.md) for
defaults, OAuth behavior, input substitution and server lifecycle.

## Theme, skill and prompt file formats

An `.itermcolors` file is an iTerm2 XML plist with named colour dictionaries,
including `Ansi 0 Color` through `Ansi 15 Color`, `Background Color`,
`Foreground Color`, and `Selection Color`. Colour dictionaries use `Red
Component`, `Green Component`, `Blue Component`, and optionally `Alpha
Component` real values. Theme lookup tries the supplied path, then `themes/`
and the config directory. The theme menu also lists `*.itermcolors` in the
user's home directory.

`SKILL.md` consists of YAML frontmatter and a Markdown instruction body:

| Frontmatter field | Type | Purpose |
| --- | --- | --- |
| `name`, `description`, `license`, `compatibility` | string | Skill identity and descriptive metadata. |
| `metadata` | map of string to string | Additional metadata. |
| `allowed-tools` | string | Space-separated tool declarations; `mcp(server-name)` starts an MCP configuration. |
| `user-invocable`, `disable-model-invocation` | boolean | Control menu/slash-command and automatic invocation. |
| `variables` | list of objects | Input fields with `name`, `label`, `description`, `default`, `type` (strings) and `options` (string list). |

See [AI Skills](AI-Skills.md) for invocation and resource rules.

`AGENTS.md` and files in `system-prompts/` are plain text/Markdown, not YAML
schemas. TTYphoon appends discovered `AGENTS.md` content to AI instructions;
system-prompt files are selected by name by the terminal AI integration.

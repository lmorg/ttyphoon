# syntaxcompletion

Config-driven syntax completion engine for editor keystrokes.

This package loads language rules from YAML and returns deterministic edit operations with caret placement.

## Config File

The embedded default config is in [default_rules.yaml](default_rules.yaml).

Top-level schema:

```yaml
languages:
  <language-name>:
    context_guard: {}
    pairs: []
    list_continuation: {}
    auto_close_tags: {}
```

Notes:
- Language names are matched case-insensitively by the engine.
- Unknown YAML keys are rejected (`yaml.Decoder.KnownFields(true)`).

## Supported Directives

### `languages`

Map of language id to language-specific rule block.

Example:

```yaml
languages:
  go:
    pairs:
      - open: "["
        close: "]"
```

### `pairs`

List of pair-completion rules for trigger characters.

Fields:
- `open` (string, required): the typed trigger.
- `close` (string, required): the inserted closing text.
- `skip_close` (bool, optional): if `true` and the immediate next source text already equals `close`, no text is inserted and the caret moves forward by `len(close)`.
- `wrap_selection` (bool, optional): reserved for future behavior. Current engine wraps active selection whenever a matching pair rule is applied, regardless of this flag.

Behavior:
- Trigger must match exactly (for example `open: "["` triggers only on `[`).
- With selection: replaces selection with `open + selection + close`.
- Without selection: inserts `open + close` and places caret between them.

Example:

```yaml
languages:
  json:
    pairs:
      - open: "{"
        close: "}"
      - open: "\""
        close: "\""
        skip_close: true
```

### `context_guard`

Syntax-context guard used to suppress completions in disallowed regions.

Fields:
- `enabled` (bool): enables syntax context checks.
- `disallow_in` (string array): contexts where completions should be blocked. Supported values are `comment` and `string` (case-insensitive).

Behavior:
- Checks run before pair/list/tag completion logic.
- Current implementation uses tree-sitter when cgo is enabled and a parser is available for the language.
- If tree-sitter is unavailable (for example non-cgo builds) or a language parser is not configured, the guard is effectively skipped.
- On classifier errors, typing is fail-open (completion is not blocked due to guard failure).

Example:

```yaml
languages:
  go:
    context_guard:
      enabled: true
      disallow_in: ["comment", "string"]
    pairs:
      - open: "["
        close: "]"
```

### `list_continuation`

Markdown-style Enter behavior for list items.

Fields:
- `enabled` (bool): enables list continuation for newline trigger (`"\n"`).
- `unordered_markers` (string array): allowed unordered markers (for example `-`, `*`, `+`). If empty, any marker matched by engine regex is allowed.
- `ordered` (bool): enables ordered list continuation (`1.`, `2)`, etc).
- `exit_on_empty_item` (bool): if current list item body is empty, pressing Enter exits the list and keeps indentation only.

Behavior constraints:
- Only runs when trigger is newline (`"\n"`).
- Only runs when caret is at end of current line.
- Current line must match supported list patterns.

Example:

```yaml
languages:
  markdown:
    list_continuation:
      enabled: true
      unordered_markers: ["-", "*", "+"]
      ordered: true
      exit_on_empty_item: true
```

### `auto_close_tags`

HTML tag auto-close behavior for `>` trigger.

Fields:
- `enabled` (bool): enables auto closing for HTML-like open tags.
- `allowed` (string array): whitelist of tag names (case-insensitive). If empty, all detected open tags are allowed.

Behavior:
- Only runs when trigger is `">"`.
- Detects open tag pattern immediately to the left of caret.
- Does not apply for closing tags (for example `</p>`) or self-closing tags (for example `<br/>`).
- Inserts `</tag>` at caret and keeps caret before the inserted close tag.

Example:

```yaml
languages:
  html:
    auto_close_tags:
      enabled: true
      allowed: ["p", "div", "span"]
```

## Runtime Trigger Mapping

Current trigger-to-rule mapping in engine:
- Any trigger string: matched against `pairs[*].open`.
- `"\n"`: used for `list_continuation`.
- `">"`: used for `auto_close_tags`.

Context guards run independently of trigger mapping and can suppress all completion kinds.

## Request/Result Contract

`Request` describes source state before the trigger is applied by the editor adapter:

- `Language`: language id.
- `Source`: full document/source text.
- `Cursor`: caret byte index in `Source`.
- `SelectionStart`, `SelectionEnd`: byte indexes. If either is `< 0`, no selection is assumed.
- `Trigger`: typed key/text (for example `[`, `\n`, `>`).

`Result` describes a single replacement edit:

- `Applied`: true if a rule matched.
- `Start`, `End`: replacement range in original source.
- `Text`: replacement text.
- `Cursor`: new caret byte index after edit.

Use `Apply(source, result)` to apply and validate the edit.

## Default Rules Included

The embedded defaults currently include:
- `go`: bracket/brace/quote pairs.
- `json`: brace/bracket/quote pairs.
- `markdown`: pair rules + list continuation.
- `html`: quote pairs + tag auto-close.

Tree-sitter context guard parsers are currently wired for:
- `bash`
- `c`
- `cpp`
- `csharp`
- `css`
- `go`
- `html`
- `java`
- `javascript`
- `json`
- `julia`
- `ocaml`
- `php`
- `python`
- `ruby`
- `rust`
- `scala`
- `typescript`
- `verilog`

Common aliases are also recognized (for example `c++`, `c#`, `js`, `ts`, `tsx`, `py`, `sh`, `shell`).

Runtime introspection helpers:
- `SupportedTreeSitterLanguages() []string`: reports grammar ids compiled into the default backend for the current build.
- `(*Engine).SupportedTreeSitterLanguages() []string`: reports grammar ids available to that engine instance.

In non-cgo builds these functions return an empty list.

See [default_rules.yaml](default_rules.yaml) for exact defaults.

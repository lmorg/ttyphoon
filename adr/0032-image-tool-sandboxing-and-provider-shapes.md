# 0032 - Image tool sandboxing, and OpenRouter's different edit shape

## Status

Accepted.

## Context

`generateImage` gained support for editing an existing image (`inputImage`) in
addition to generating a new one. Two problems surfaced once this was used for
real:

1. **OpenRouter has no `/images/edits` endpoint.** OpenAI's Images API is two
   endpoints: `POST /images/generations` (JSON) and `POST /images/edits`
   (multipart file upload). OpenRouter implements neither name directly; its
   dedicated Images API takes a single request shape where an existing image
   is passed as `input_references` (inline data/HTTP URLs) alongside the
   prompt, on the same request used for generation. Posting a multipart
   request to `<base>/images/edits` against OpenRouter 404s, since that path
   simply doesn't exist there - it isn't a bug in the request content, the
   endpoint itself is the wrong shape for that provider.

2. **The model can't browse where it already writes.** `generateImage` writes
   to `~/ttyphoon/.images/` by default, deliberately outside the project root
   so generated images don't clutter the user's workspace (see ADR 0029). But
   `readDirectory`, like every other file tool, rejected any path outside the
   project root. The model had no way to list what it had already generated in
   order to reference a file for editing, other than remembering the exact
   path it was told at generation time.

## Decision

**Branch the HTTP shape by provider, not by feature.** `isOpenRouterBaseURL`
detects OpenRouter from the configured base URL. When editing against
OpenRouter, the request goes through the same JSON generation path as a normal
call, with the input image attached as a base64 `data:` URL under
`input_references`. Every other provider (real OpenAI, and anything else
OpenAI-compatible) keeps the original multipart `/images/edits` call. There is
no generic "try both, fall back on 404" logic: a 404 from a misconfigured
provider should surface as an error, not be silently retried with a different
request shape.

**`resolveInputImagePath` trusts two roots, not one.** An `inputImage` must
resolve inside the project root *or* inside `~/<app>/.images` - the same
directory `generateImage` itself writes to. This mirrors the boundary already
used for output paths (ADR 0029's default path), rather than opening the tool
up to arbitrary absolute paths.

**`readDirectory` is the one file tool allowed outside the project root.**
`ai/tools/file/path.go` now has two entry points: `resolveWorkspacePath`
(unchanged; rejects anything outside the project root) and `resolveAnyPath`
(no root restriction), both backed by the same `resolvePath(agt, name,
restrictToRoot)`. Only `Directory.Call` uses `resolveAnyPath`. `readFiles`,
`write`, `patch`, and `insert` are unchanged and still workspace-restricted,
since those can exfiltrate or modify arbitrary files, whereas listing a
directory's names is low-risk and is what actually unblocks discovering
generated images.

## Consequences

- Editing an existing image works against both OpenAI and OpenRouter, using
  the request shape each one actually implements.
- `readDirectory` can list `~/ttyphoon/.images` (or any other absolute path)
  but cannot read file contents or write/patch outside the workspace - those
  tools kept their existing restriction.
- Adding a third provider with yet another edit shape means adding another
  branch here, not a generic abstraction; two providers isn't enough evidence
  to design that abstraction yet.

## References

- `ai/tools/image/generate.go` - `isOpenRouterBaseURL`, `requestImage`,
  `requestImageEdit`, `resolveInputImagePath`, `encodeImageDataURL`
- `ai/tools/file/path.go` - `resolveWorkspacePath`, `resolveAnyPath`,
  `resolvePath`
- `ai/tools/file/directory.go` - `Directory.Call`
- ADR 0029 (amended in spirit): the `~/ttyphoon/.images` boundary is now
  referenced from two places, not just the writer

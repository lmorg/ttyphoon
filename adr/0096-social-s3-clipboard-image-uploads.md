# 0096 - Social S3 uploads provide shareable clipboard image URLs

Date: 2026-11-10

## Status

Accepted and implemented.

## Context

The rendered-selection clipboard pipeline (ADR 0095) can embed PNGs in HTML and
RTF, but Markdown needs an image URL that the receiving application can access.
Local paths and generated diagram data URLs do not provide that. Slack may also
ignore embedded images, so its custom clipboard representation uses numbered
figure references for images with HTTP(S) sources.

An optional upload destination lets copied local images, charts, Mermaid
diagrams, and other generated graphics acquire shareable URLs. This is a separate
architectural decision from formatting: it introduces credentials, external
network requests, persistent objects, and a deliberate data-publication boundary.

## Decision

### Package and configuration

`utils/s3` owns configuration validation, PNG upload, S3 authentication, and
public URL construction. It does not depend on Wails or frontend state.

Settings live exclusively under `config.Config.Social.S3`, represented by a
pointer to `config.S3Config`. An absent/null S3 section disables uploads. A
present but incomplete section is an error, not an implicit opt-out.

```yaml
Social:
  S3:
    Endpoint: "" # optional S3-compatible HTTPS endpoint
    Region: us-east-1
    AccessKeyID: "YOUR_ACCESS_KEY_ID"
    SecretAccessKey: "YOUR_SECRET_ACCESS_KEY"
    SessionToken: "" # optional temporary-credential token
    Bucket: my-bucket
    Prefix: clipboard/images
    PublicBaseURL: https://cdn.example.com
```

| Field | Required | Meaning |
|---|---|---|
| Endpoint | No | HTTPS service root, optionally including a path; defaults to `https://s3.<Region>.amazonaws.com` |
| Region | Yes | SigV4 signing region, including for custom endpoints |
| AccessKeyID | Yes | Explicit signing credential |
| SecretAccessKey | Yes | Explicit signing secret |
| SessionToken | No | Token for temporary credentials |
| Bucket | Yes | Bucket name, not a bucket path or URL |
| Prefix | No | Object-key directory within the bucket |
| PublicBaseURL | Yes | Public bucket/CDN root, excluding Prefix |

All fields are literal configuration values: the constructor does not expand
environment references or populate credentials from host environment variables.
The uploader does not use AWS profiles, instance metadata, or the default
credential-provider chain. Protect credential-bearing configuration files and
keep them out of source control. No credentials are sent to the frontend.

Endpoint and PublicBaseURL must use HTTPS and cannot contain URL credentials,
queries, or fragments. Prefix rejects traversal segments and unsafe path
characters; leading/trailing slashes are removed. A custom endpoint must support
path-style S3 requests.

### Object naming, paths, and public access

Each successful upload uses a fresh random UUIDv4 filename ending in `.png`.
This is a random identifier, not a content hash.

For the configuration above, an object is written to:

```text
https://s3.us-east-1.amazonaws.com/my-bucket/clipboard/images/<uuid>.png
```

Its published reference is:

```text
https://cdn.example.com/clipboard/images/<uuid>.png
```

Bucket is appended only to the upload endpoint. Prefix is appended to both the
upload bucket path and PublicBaseURL. The public URL is constructed, not probed:
the user must configure the bucket/CDN mapping and public read policy correctly.

Requests do not set a public ACL or change bucket policy. Credentials should
have only the necessary `s3:PutObject` permission on the configured prefix;
the application does not need list or delete permissions for this feature.
Public reading is the user's infrastructure responsibility. Random filenames
reduce accidental collisions but are not access control.

URLs are permanent rather than presigned. There is no expiry, deletion,
cross-copy object reuse, or automatic cleanup. Use bucket lifecycle rules for
retention where appropriate, bearing in mind that deleting objects breaks
previously pasted links.

### Upload protocol and limits

Use the existing AWS SDK v2 SigV4 signer rather than implementing authentication
or adding an additional S3 client dependency. Upload with one HTTPS PUT:

- Service name `s3`, configured signing region, and explicit credentials.
- `Content-Type: image/png`.
- SHA-256 payload digest and `X-Amz-Content-Sha256`.
- Session-token signing when configured.
- Path-style bucket addressing, preserving endpoint base paths.
- A 30-second HTTP client timeout per request.
- No following redirects, to avoid forwarding a signed upload to another URL.
- HTTP 200 is required for success; other statuses become explicit errors.

There are no automatic retries or multipart uploads. PNG header/config validation
requires positive dimensions, at most 16384 pixels per side, at most 16 Mi
pixels total, and at most 32 MiB of encoded PNG data. This validation is not a
full image decode. The complete copy input also retains ADR 0095's size and
nesting limits.

Connection errors produce a bounded warning without echoing endpoint URLs or
credentials. HTTP failures report the status code rather than the service's
response body.

### Clipboard integration

1. The frontend captures the selected/live rendered content. Its existing
   rasterization converts SVG, Mermaid, charts, canvas, and ordinary images to
   inline PNGs before invoking Go.
   SVG snapshots use their intrinsic viewBox dimensions (or explicit pixel
   dimensions before layout bounds), not the scaled AI-panel/view size, and
   remove root sizing constraints before rasterization. Bitmap images use
   natural dimensions; canvas snapshots use their backing-store dimensions.
   Existing image-size limits still apply rather than silently downscaling.
2. `CopyFormattedSelection` and `CopyFormattedTable` obtain an optional uploader
   from Social/S3. This also covers AI Copy prompt/response, since those actions
   use the shared rendered copier.
3. `copy_format.ConvertWithImageUpload` validates and sanitizes the entire
   fragment before invoking any uploads.
4. Images whose original source is already HTTP(S) are left alone. Local,
   generated, data-URL, or otherwise non-HTTP(S) sources are upload candidates.
5. Uploads run sequentially. Identical encoded PNG sources within a single copy
   reuse a successful upload. This cache is not persisted across copy requests.
6. The uploaded URL replaces internal image-source metadata used by text
   serializers. It does not replace the inline PNG in standard HTML or RTF.
7. Publish the completed representations together in one clipboard write.

| Representation | Uploaded-image behavior |
|---|---|
| Standard HTML and Chromium `text/html` | Keep base64 inline PNGs |
| RTF | Keep embedded PNG picture data |
| Selection plain text and Chromium `text/markdown` | Use Markdown image references to the public URL |
| Formatted-table plain text | Retain existing tab-delimited displayed-value rules |
| Chromium `slack/html` | Use `[fig N: URL]` with blank-line spacing, including structured table-cell content |

Figure references increment in document order per clipboard conversion.
Internal `data-copy-source` attributes are removed from both exported HTML
representations after their values have been used.

`copy_format.Convert` remains network-free. `ConvertWithImageUpload` receives
an explicit upload callback, keeping the converter independent of configuration
and S3. Legacy direct HTML/formatted clipboard setters do not initiate uploads.

### Failure behavior and user consent

Configuring Social/S3 explicitly opts into sending local/generated copied images
to that service. There is no per-copy confirmation. With no configuration,
copying performs no S3 requests and retains its previous inline-image behavior.

The user selected a best-effort copy policy:

- An upload or applicable configuration failure replaces the affected image
  with its alt text, or `Image` if no alt text exists.
- Remaining text, tables, and unaffected images are still copied.
- Go returns warning strings after a successful clipboard write. The shared
  frontend copier displays them as an image-fallback warning.
- Clipboard-write or conversion failures remain errors rather than successful
  fallback copies.

The copy methods therefore return warning lists plus an error; their generated
Wails bindings must stay synchronized (ADR 0013). Frontend rasterization failures
also continue to use ADR 0095's warning/fallback path.

Uploads are not transactional with each other or the clipboard write. An object
already uploaded is not removed if a subsequent upload or clipboard write fails.
Cancellation/timeout can also leave an object whose response was not received.
There is no document mutation: only the export snapshot's image references change.

## Consequences

- Markdown and Slack can reference images from copied local/rendered content
  without relying on machine-local file paths or data-URL support.
- Copying may now transfer sensitive image contents outside the machine when
  explicitly configured. Users must choose an appropriate service, public-read
  policy, credentials, and retention policy.
- Local rich copies become network-dependent and may take up to the per-request
  timeout for each attempted upload. A stalled request delays subsequent queued
  copies; there is no separate overall-copy deadline or cancellation UI.
- Existing remote images do not generate additional stored objects. Repeated
  copies of local images do, because UUID names are generated per copy.
- Successful PUT does not prove that Slack or another receiver can fetch the
  public URL. CDN mapping, access policies, propagation, and receiver behavior
  remain deployment/integration concerns.
- Inline HTML/RTF remain self-contained after successful uploads. On upload
  failure, the selected fallback policy removes that image from all formats and
  uses alt text instead.

## Verification

- Config tests decode every Social/S3 field with strict YAML field checking.
- Uploader tests use a local TLS HTTP server to verify PUT bytes, content type,
  digest, credential/signing scope, session token, UUID filenames, upload paths,
  public URLs, and the absence of a public ACL.
- Tests cover omitted/incomplete configuration, literal values without environment expansion, invalid
  URLs/paths, non-PNG input, HTTP failures, and redirect rejection.
- Converter tests cover successful URL propagation, per-copy deduplication,
  skipping existing HTTP(S) sources, table figure references, inline HTML/RTF
  preservation, alt-text warnings, metadata stripping, and complete validation
  before any upload.
- Frontend tests verify that upload warnings returned by Go are notified by the
  shared copier. Copy bindings are regenerated and the frontend build checked.
- No real bucket upload or public CDN fetch was performed during implementation.
  Real endpoint credentials, permissions, URL accessibility, and receiver paste
  behavior require deployment verification.

## References

- [ADR 0095: Rendered selection clipboard formats](0095-formatted-rendered-selection-copy.md)
- [ADR 0013: Wails bindings](0013-wails-binding-checklist.md)
- [S3 uploader](../utils/s3/upload.go) and [tests](../utils/s3/upload_test.go)
- [Configuration schema](../config/social.go) and [example](../config/defaults.yaml)
- [Image conversion/upload integration](../utils/copy_format/copy.go)
- [Clipboard binding boundary](../window/frontend/main/clipboard.go)
- [Frontend copy queue and warnings](../frontend/src/copy-formatted.js)

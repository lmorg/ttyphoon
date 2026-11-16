# AI Web Search

| Environment Variable | Default | Meaning |
| --- | --- | --- |
| `TTYPHOON_WEB_SEARCH_PROVIDER` | `duckduckgo` | Select `duckduckgo` or `searxng`. Unset or empty preserves DuckDuckGo. |
| `TTYPHOON_SEARXNG_URL` | Unset | Required for `searxng`: an absolute HTTP(S) instance URL, optionally with a path prefix or `/search`. Credentials, query parameters, and fragments are not allowed. |
| `TTYPHOON_SEARXNG_HEADERS` | No extra headers | Optional JSON object of header names to string values, up to 32 KiB. Supports private-instance authentication; requires HTTPS except on loopback. Invalid or reserved headers are rejected. |
| `TTYPHOON_SEARXNG_MAX_CONCURRENT` | `2` | Integer from `1` to `16`. Shared limit for active search and discovery requests to the same instance across agents. |

These settings come from AI-wide `EnvFiles` over the application launch
environment, not the selected service or active terminal pane. Existing search
tools pick up new file references when settings are reloaded. The three
`TTYPHOON_SEARXNG_*` settings are ignored when the provider is DuckDuckGo.

The existing web-search tool defaults to DuckDuckGo's Instant Answer API. To
use a SearXNG instance instead, set these variables in the environment that
launches TTYphoon:

```sh
export TTYPHOON_WEB_SEARCH_PROVIDER=searxng
export TTYPHOON_SEARXNG_URL=http://127.0.0.1:8080
```

Alternatively, an AI-wide file can contain these literal lines (without
`export` or shell quoting):

```text
TTYPHOON_WEB_SEARCH_PROVIDER=searxng
TTYPHOON_SEARXNG_URL=http://127.0.0.1:8080
TTYPHOON_SEARXNG_HEADERS={"Authorization":"Bearer REPLACE_WITH_TOKEN"}
```

Private instances can use authentication headers configured in an AI-wide file
or the launch environment, never in AI tool arguments:

```sh
export TTYPHOON_SEARXNG_HEADERS='{"Authorization":"Bearer REPLACE_WITH_TOKEN"}'
export TTYPHOON_SEARXNG_MAX_CONCURRENT=2
```

The headers variable is a JSON object; custom authentication headers such as
`X-API-Key`, Basic authorization, and cookies are also supported. Keep real
values in a private file, launcher environment or secret manager, not source
control or tool prompts. Credential-bearing URLs and reserved routing/framing headers
are rejected. Authentication requires HTTPS except for loopback instances.
Configured credential values are redacted from tool output and errors, and are
never applied to DuckDuckGo or forwarded through redirects. Changing the shared
concurrency limit for an already initialized instance requires a restart.

SearXNG search and discovery share a per-instance concurrency limit across
agents: two requests by default, configurable from 1 to 16. The slot remains
held until the response body is closed; queued requests can be cancelled. HTTP
429 and 503 responses start a shared cooldown using `Retry-After` seconds or an
HTTP date. A missing or invalid value uses 30 seconds. During cooldown, new
outbound requests fail promptly with the remaining delay and UTC retry time;
they are not automatically retried or sent to another provider. Existing
in-flight requests are not aborted, and cached discovery remains usable.

The URL may include an instance path prefix or end in `/search`. The instance
must permit JSON responses in its `settings.yml`:

```yaml
search:
  formats:
    - html
    - json
```

Many public instances disable JSON or rate-limit automated requests; a private
instance is preferable. Searches return titles, URLs, and plain-text snippets,
not complete pages. The existing `scrapeWebPage` tool can read a selected page.
The tool is exposed as `webSearch` regardless of the configured provider.

SearXNG accepts optional categories, engines, language, and time-range filters,
plus bounded pagination and result limits:

```json
{"query":"Go release notes","categories":["general"],"language":"en","time_range":"month","page":2,"limit":5}
```

`page` defaults to 1 and accepts 1-10; `limit` defaults to 10 and accepts 1-50.
The limit applies to unique URL results returned to the AI, not an upstream
engine's page size. Additional pages are never fetched automatically. DuckDuckGo
also accepts `limit`, but its Instant Answer API does not support later pages.

Results preserve publication dates, publishers/source labels, and contributing
engines when supplied. Missing dates are reported as unknown. Known tracking
parameters (`utm_*`, `fbclid`, `gclid`, `msclkid`) are removed for URL
deduplication; meaningful query parameters and fragments are retained. Duplicate
URLs merge engine provenance, not independent corroboration.

SearXNG may also return suggested queries, corrections, direct answers, and
source-linked infobox summaries. These are bounded supplements, not automatic
follow-up actions. Answers lacking attribution say so; infoboxes without a valid
source URL are omitted. Search output is capped at 64 KiB, with a truncation
notice when necessary. Image metadata and image downloading are not included.

To discover your instance's engine names, shortcuts, categories, and support
for paging, language filters, time ranges, and safe search, call the same tool
without a search query:

```json
{"action":"discover"}
```

Discovery uses the instance's `/config` endpoint. Results are cached for ten
minutes per search-tool instance; use `{"action":"discover","refresh":true}`
to reload them. Engine `enabled` means enabled by default, not current engine
health. Missing capability flags remain unknown. Discovery does not send a
query to upstream search engines, and an inaccessible `/config` endpoint does
not prevent normal searches. Normal searches do not fetch discovery metadata
automatically. Large responses report their full engine count and whether the
returned metadata was truncated.

When fresh discovery metadata is cached, unknown engine/category names are
rejected with a refresh hint. Known engine shortcuts are resolved to full names.
Truncated caches cannot definitively reject an unlisted name, and missing or
expired discovery never blocks searching. Advertised paging, language,
time-range, default-enablement, and category mismatches produce advisories rather
than silently changing the request. This validation does not fetch `/config`.

Use category and engine names supported by your instance. Time ranges are
`day`, `month`, or `year`; support varies by upstream engine. SearXNG errors do
not silently fall back to another provider, and search redirects are not
followed. To restore DuckDuckGo, unset `TTYPHOON_WEB_SEARCH_PROVIDER` or set it
to `duckduckgo`. These optional filters require the SearXNG backend.

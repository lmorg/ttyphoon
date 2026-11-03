# 73. Web search supports configurable DuckDuckGo and SearXNG backends

Date: 2026-10-16

## Status

Accepted.

## Context

The native web-search tool uses DuckDuckGo's Instant Answer API, which is not
a general search-results API. A configurable SearXNG instance provides
metasearch results and engine/category selection without adding another tool
or a provider-specific model integration.

## Decision

- Own web search in `ai/tools/websearch`, with separate tool dispatch,
  configuration, DuckDuckGo, SearXNG, metadata discovery, and text-processing
  files. Keep page scraping in `ai/tools/webtools`. Register both packages
  through `ai/tools/tools.go`, retaining tool names and permissions. Move all
  existing search tests with their implementation rather than duplicating them
  in the scraping package.
- Reuse the existing structured search tool and retain its `searchDuckDuckGo`
  identifier, enabled state, and default permissions. Saved permissions and
  invocation histories must not silently move to a different tool identity.
  Its description identifies the configured provider. `scrapeWebPage` remains
  a separate tool; searching does not fetch result URLs automatically.
- Read `TTYPHOON_WEB_SEARCH_PROVIDER` when constructing the tool. Empty or
  `duckduckgo` preserves the existing default. `searxng` requires
  `TTYPHOON_SEARXNG_URL`, an HTTP(S) instance URL or search endpoint. Preserve
  instance path prefixes and append `/search` when needed. Reject unknown
  providers, invalid URLs, and URLs containing a query or fragment. Setup
  failures use the existing tool-initialization error path; there is no silent
  cross-provider fallback. These variables belong to the application's launch
  environment, not the active terminal pane's environment.
- Read optional `TTYPHOON_SEARXNG_HEADERS` as a JSON object of header names to
  string values, limited to 32 KiB, when constructing the tool. Use it for
  Authorization, API-key, cookie, or private-proxy authentication on both search
  and discovery. Reject invalid names/values, duplicate case-insensitive names,
  routing/framing headers, and overrides of the tool's Accept/User-Agent headers.
  Reject URL userinfo and require HTTPS for authenticated non-loopback instances.
  These credentials are never model inputs or tool-description fields and are
  never applied to DuckDuckGo. Redirects remain disabled.
- Centralize SearXNG requests in `transport.go`. Redact configured header values,
  authorization payloads, parsed Basic credentials, and cookie values from
  returned output and errors, including JSON-escaped echoes. Invalid credential
  configuration errors do not print supplied values. Transport errors omit the
  request URL and preserve error unwrapping/cancellation through a sanitized
  error type. Tool observations receive redacted output/errors, not headers.
- Share one in-memory request gate per configured instance endpoint across tool
  and agent instances. `TTYPHOON_SEARXNG_MAX_CONCURRENT` defaults to 2 and accepts
  1-16. Keep a slot until the response body closes, release it on errors, and let
  queued callers cancel through their invocation context. A different limit for
  an already initialized endpoint requires restarting the application.
- HTTP 429 and 503 establish a shared cooldown using `Retry-After` delta seconds
  or an HTTP date. Invalid/missing values default to thirty seconds; past dates
  mean no further delay. Parse delta seconds without duration overflow and keep
  the latest deadline when concurrent responses disagree. Check cooldown both
  before queuing and after acquiring a slot. Fail new requests promptly with
  remaining seconds and a UTC retry time instead of sleeping, retrying
  automatically, or changing providers. Existing in-flight requests may finish;
  cached discovery does not require an outbound slot. Cooldowns are not persisted.
- Request SearXNG JSON with GET and URL-encoded `q` and `format=json` parameters.
  The administrator must enable `json` under `search.formats`. Explain HTTP 403
  as a possible JSON-format or access-policy issue. Preserve HTTP status errors,
  invalid responses, network errors, and caller cancellation for the existing
  tool recovery/observation boundary (ADR 0001).
- Search accepts `query` with no action or with `action:"search"`. SearXNG
  additionally accepts optional `categories` and `engines` arrays, `language`,
  and `time_range` (`day`, `month`, or `year`). Names and upstream support depend
  on the instance. Reject these filters when DuckDuckGo is selected rather than
  silently ignoring them. `page` defaults to 1, accepts 1-10, and maps to SearXNG's
  `pageno`. `limit` defaults to ten and accepts 1-50 unique URL results on either
  provider. DuckDuckGo rejects pages after the first. These bounds do not imply
  an upstream page size, a known total result count, or automatic pagination.
- Add explicit `action:"discover"` requests with optional `refresh:true` and
  no query or search filters. Read the configured instance's `/config` endpoint,
  preserving any instance path prefix. Return its name/version, sorted category
  and engine identifiers, shortcuts, and advertised flags for default enablement,
  paging, language, time ranges, and safe search. Preserve absent flags as
  unknown; `enabled:false` means disabled by default, not necessarily unusable
  when explicitly selected. This metadata is not an engine-health check.
- Cache successful discovery responses for ten minutes per search-tool instance
  and configured endpoint. Coalesce concurrent misses into one in-flight
  request, let waiters cancel without blocking on a mutex-held HTTP call, and
  permit explicit refresh. Do not cache failures. Normal searches never require
  discovery or issue an automatic `/config` request; restricted or older
  instances can still search without it. Discovery sends no query to upstream
  engines and uses the same timeout, response-size, and no-redirect policy.
- Bound discovery to at most 256 engines and 64 KiB of JSON output, with the
  original engine count and a truncation flag. Bound individual identifiers and
  category lists, omitting invalid/oversized identifiers rather than changing
  them into different filter names. Treat metadata as untrusted data. Do not
  expose unrelated instance configuration or replace permission state from
  metadata. Discovery observations use a short summary, not the full JSON blob.
- Use fresh cached discovery, when present, to validate explicit engine/category
  names and resolve engine shortcuts without mutating the caller's input. An
  unknown name in a complete cache is an actionable error with a refresh hint;
  missing, expired, or truncated metadata must not become an implicit discovery
  dependency or an incorrect rejection. Advertised false capability flags,
  disabled-by-default explicit engines, and category mismatches produce bounded
  advisories rather than blocking a valid request. Unknown flags remain unknown.
- Keep a 20-second HTTP timeout, limit decoded provider response input to 2 MiB,
  and apply the requested bound to unique valid HTTP(S) result entries. Limit titles to 300
  characters, snippets to 2000 characters, and URLs to 4096 bytes. Strip HTML and
  scripts from display text. Cap total search output at 64 KiB without breaking
  UTF-8, with an explicit truncation notice. If engines were unavailable,
  disclose partial results or report an error when no usable results remain.
- Preserve supplied publication timestamps (`publishedDate`, or `pubdate` as a
  fallback), publishers/source labels, and engine provenance. Missing publication
  dates are unknown, never inferred from snippets or query filters. Normalize
  HTTP(S) URLs for deduplication by host case and query encoding and remove only
  known tracking keys (`utm_*`, `fbclid`, `gclid`, `msclkid`). Retain meaningful
  query values, duplicate values, paths, schemes, and fragments. Merge engine
  names for duplicate URLs while preserving the first result's position; only
  fill missing metadata from duplicates. A duplicate URL is not independent
  corroboration, and invalid or credential-bearing URLs are omitted.
- Include up to five distinct suggestions and five corrections, explicitly
  labelled as not executed. Include up to three direct answers (both legacy
  strings and current structured answer objects), retaining valid source URLs
  and marking absent attribution. Include at most three infoboxes with valid
  HTTP(S) source links; omit unsourced summaries. Bound infobox text, retain up
  to eight scalar attributes and three source links, and ignore image metadata.
  Supplemental results do not consume the URL-result limit, but do share the
  overall output bound. Do not automatically search suggestions or fetch links.
- Treat all returned search content as untrusted external data, not agent
  instructions. HTML stripping is presentation cleanup, not a prompt-injection
  security boundary. Do not follow search redirects, including SearXNG's
  external/automatic `!!` bangs. This keeps a search from unexpectedly becoming
  a direct request to an external page. A private SearXNG instance still sends
  queries to its configured upstream engines; it is not an anonymity guarantee.
- Test using local HTTP fixtures and a synthetic DuckDuckGo transport, never a
  live public search service. Cover configuration, path prefixes, query/filter
  encoding, result bounds, HTML cleanup, empty/direct answers, partial failures,
  HTTP errors, disabled JSON, redirects, cancellation, and the unchanged default
  DuckDuckGo request contract. Discovery fixtures also cover cache hits, expiry,
  explicit refresh, independent tool caches, concurrent callers, capability
  flags, exact identifiers, truncation, and search after discovery failure.
  Rich-result fixtures cover optional metadata, both answer shapes, attributed
  infoboxes, ignored images, one-request pagination, input/output caps, URL
  deduplication, and advisory cached validation without hidden network requests.
  Transport fixtures use synthetic credentials only and cover authenticated
  search/discovery, credential redaction, redirect isolation, DuckDuckGo
  separation, shared cooldown/expiry, cancellable concurrency, and Retry-After
  date/seconds parsing without sleeping or contacting a real instance.

## Consequences

Existing users retain their search tool and permissions without configuration
changes. SearXNG users get richer search results through an instance they
control, with predictable errors instead of accidental query disclosure to a
fallback provider. The legacy tool identifier is intentionally retained even
though its backend is configurable.

Instance discovery and richer search results remain part of the existing tool
rather than separate permission identities. News metadata, refinement hints,
bounded pagination, deduplication, and attributed summaries improve research
without automatic query chains or a mandatory discovery request. Image support
is deferred. Full-page retrieval continues through `scrapeWebPage`.

## Reference

- `ai/tools/websearch/tool.go`
- `ai/tools/websearch/config.go`
- `ai/tools/websearch/duckduckgo.go`
- `ai/tools/websearch/searxng.go`
- `ai/tools/websearch/discovery.go`
- `ai/tools/websearch/filters.go`
- `ai/tools/websearch/results.go`
- `ai/tools/websearch/refinements.go`
- `ai/tools/websearch/transport.go`
- `ai/tools/websearch/ratelimit.go`
- `ai/tools/websearch/web_search_test.go`
- `ai/tools/websearch/discovery_test.go`
- `ai/tools/websearch/filters_test.go`
- `ai/tools/websearch/results_test.go`
- `ai/tools/websearch/transport_test.go`
- `ai/tools/webtools/web_tools.go` - separate page scraper
- `README.md` - AI Web Search setup
- [SearXNG Search API](https://docs.searxng.org/dev/search_api.html)
- [SearXNG search syntax](https://docs.searxng.org/user/search-syntax.html)
- [SearXNG webapp source](https://github.com/searxng/searxng/blob/master/searx/webapp.py) - `/config` response fields
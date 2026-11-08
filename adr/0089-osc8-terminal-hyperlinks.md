# 0089 - OSC 8 terminal hyperlinks reuse Go hyperlink actions

Date: 2026-11-04

## Status

Accepted and implemented

## Context

Applications can explicitly attach a URI to arbitrary terminal text using
`OSC 8 ; parameters ; URI ST`, closing the link with an empty URI. Automatic
URL detection cannot recover that target, and converting the label into a
single drawable element would interfere with ordinary terminal styles,
overwrites, wide characters, and reflow.

## Decision

- Gate support on `Terminal.OSC8Hyperlinks`. Its zero value is false; the
  shipped `defaults.yaml` enables it. Users can explicitly set it to false.
  This setting is independent of `Terminal.AutoHyperlink`.
- Consume OSC 8 with either BEL or ST termination, even when disabled. Split
  the payload at only the first two semicolons so URI semicolons survive.
  Accept optional parameters without interpreting them; `id` does not merge
  separate open sequences into one hover group.
- An opener replaces the active target; an empty URI closes it. Invalid
  openers clear the previous target and log a warning.
- Use the existing `Cell.Element` reference with a shared `types.TextElement`
  wrapper, rather than a separate hyperlink field. This mode retains original
  characters and SGR while delegating interactions to the wrapped element.
  Ordinary mxAPC elements still capture text and draw using element coordinates;
  text-preserving elements use normal cell/ligature rendering. Selection,
  erasure, scrolling, and resize reflow retain their normal behaviour. Hit
  testing resolves the second column of a wide character to its leading cell.
- Track the active text-preserving element per terminal, independently of
  mxAPC's capturing element. Captured content and inserted drawable elements
  are not annotated; subsequent ordinary text resumes the active annotation.
- Use the existing Go `element_hyperlink` implementation for labels, targets,
  tooltips, clicks, and action menus through the shared element dispatch.
  Accept URI schemes without `://`, such
  as `mailto:`, in both that element and the shared hyperlink menu parser.
  Keep auto-detected file paths containing unescaped spaces or percent signs
  working.
- HTTP(S) tooltips parse the URL authority and show a literal IP address,
  non-empty username, and explicitly supplied password only when present.
  Use icons `f1eb`, `f007`, and `f084` respectively and HTML-escape values.
  Do not resolve hostname addresses or infer credentials from query parameters.
- HTTP(S) tooltips show `/` for an absent path, a hostname (`f015`) only for
  non-IP hosts, and a port (`f13d`) only when explicitly present in the URL.
- HTTP(S) tooltip rows appear in this order: Scheme, Hostname or IP, Port,
  Path, User, Pass, then query parameters. Omit absent optional fields without
  changing the order of remaining rows. File tooltips show Path then File;
  other schemes and malformed HTTP(S) targets show Scheme then Path.
- Render hyperlink tooltip HTML once during element generation and reuse the
  cached string on hover. Keep rendering in `utils/hyperlink_tooltip` without
  renderer or element dependencies; Notes HTTP(S) and file-link hovers use the
  same renderer through `GetHyperlinkTooltip`.
- Highlight the actual surviving linked cells, rather than redrawing the
  original label. Underline-only hover uses a rendering copy of SGR and does
  not change application-provided styles.
- Do not let automatic URL/file detection replace explicit targets or overlay
  them with a drawable auto-link label.
- Reset the active target on terminal reset and normal/alternate buffer
  transitions. Existing cells retain their targets until overwritten or erased.
  Mouse-reporting applications retain precedence over hyperlink interaction.
- Declare tmux's `hyperlinks` feature for this client with `-T` only when
  support is enabled, without forcing the feature for other attached clients.
  The existing control-mode output decoder carries the OSC bytes unchanged;
  OSC 8 requires no query response, passthrough wrapper, or identity change.

## Consequences

- Application-provided labels can differ from their destinations. Hover shows
  the actual target; opening still requires the existing explicit user action.
- Disabling OSC 8 ignores supplied targets without exposing escape bytes or
  disabling ordinary auto-detection.
- Targets follow in-memory terminal cells, including scrollback. This does not
  add hyperlink persistence to tmux screen captures or plain-text disk history.
- Applications may still require `--hyperlink=always`; capability heuristics
  and a new terminfo identity remain governed by ADR 0080.

## Validation

Regression coverage includes open/close/replacement, configuration, parameters,
URI schemes and semicolons, BEL/ST, byte-split streams, Unicode width, wrapping,
SGR, hover, existing menus, copying, overwrites, erasure, scrollback, reflow,
buffer transitions, automatic detection precedence, and tmux output decoding.

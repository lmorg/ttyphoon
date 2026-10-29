# 0080 - Reporting terminal capabilities

Date: 2026-10-28

## Status

Accepted as capability-reporting and investigation guidance. This record does
not implement OSC 8, Kitty graphics, a new terminfo entry, or changes to tmux's
response handling.

## Context

Ttyphoon may implement protocols also supported by Kitty and Ghostty, such as
OSC 8 hyperlinks and the Kitty graphics protocol. Applications must discover
support without being misled into assuming the entire feature set of another
terminal emulator.

There is no single universal capability declaration. Applications combine
environment variables, terminfo, protocol-specific queries, and implementation
allowlists. Ttyphoon also renders pane output received through tmux control
mode, while tmux maintains its own parser and can answer application queries
independently. Seeing raw output is not the same as owning all query responses.

## Decision

### Identity and Terminfo

- Advertise the actual implementation, not Kitty or Ghostty. Do not set another
  emulator's `TERM_PROGRAM`, `TERM`, or emulator-specific environment variables
  merely to trigger application feature detection.
- `TERM` names a terminal description. A future direct-PTY identity such as
  `xterm-ttyphoon` requires a matching, installed terminfo entry. It must also be
  available on remote hosts where applications use it; otherwise provide a
  deliberate compatibility fallback rather than an unknown terminal name.
- In ordinary tmux panes, the terminal description must remain tmux-compatible,
  for example `tmux-256color`. An outer emulator's terminfo entry does not make
  the inner tmux parser implement all of the emulator's protocols.
- `TERM_PROGRAM` and `TERM_PROGRAM_VERSION` can identify the emulator/version,
  but are not general feature guarantees. Some application libraries need their
  allowlists updated to recognize a new identity.
- `COLORTERM=truecolor` describes colour depth, not OSC 8 or graphics support.
- Current `config/env.go` exports `TERM=xterm-256color`, the legacy identity
  `TERM_PROGRAM=mxtty`, and `MXTTY_VERSION`. These are current code facts, not a
  claim that a custom ttyphoon terminfo entry or `TERM_PROGRAM_VERSION` is already
  installed or exported. tmux can set the environment seen inside its panes.

### Feature-Specific Discovery

OSC 8 has no universally supported query/reply handshake. Clients commonly use
terminal/version heuristics, support tables, extended terminal capabilities
where understood, or explicit options such as `--hyperlink=always`. Implement
the actual opening/closing hyperlink sequences and configure tmux's hyperlink
capabilities only when the end-to-end implementation supports them. The tmux
manual documents the `hyperlinks` feature and the `Hls` terminfo extension;
neither means every application will automatically recognize a new emulator.
Ttyphoon's `sessionSetOptions` currently comments out `hyperlinks`.

Kitty graphics defines an explicit probe. In C-style escape notation, a client
can send this direct-transmission query followed by primary device attributes:

```text
\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\\x1b[c
```

A supporting implementation validates the dummy image and responds without
storing or displaying it:

```text
\x1b_Gi=31;OK\x1b\\
```

The protocol uses the response order for detection: receiving the device-
attributes reply without a preceding graphics-query reply indicates unsupported
graphics. Respond to the graphics query before processing later input. Advertise
only implemented formats, actions, and transmission media; support for direct
image data does not imply support for filesystem or shared-memory transfers,
animation, or every other protocol extension.

Provide truthful cell/pixel dimensions through the relevant PTY/window-size
interfaces and supported window-operation queries. Graphics detection and image
placement both depend on an end-to-end implementation, not just an identity
string or successful base64 decoding.

### tmux Responses Versus Raw Output

The upstream tmux manual and `input.c` reviewed on this date provide no general
documented switch that makes tmux stop all terminal parsing/query replies and
become a transparent PTY relay. Its device-attributes and device-status handlers
generate replies through `input_reply`, which writes back to the pane's PTY.
Some individual features have their own options, but these are not a global
disable-all-responses facility.

`allow-passthrough` is a different mechanism:

```tmux
set -g allow-passthrough on
```

This example documents an option, not a configuration change applied by this
ADR. With `on`, explicitly wrapped passthrough is allowed for visible panes;
`all` also permits invisible panes. The application must send the special
envelope, with each embedded ESC byte doubled:

```text
ESC P tmux; <escaped payload> ESC \
```

tmux can then forward the payload to an attached terminal without interpreting
it as ordinary pane terminal commands. This does not automatically wrap plain
CSI/OSC/APC sequences, stop tmux parsing everything else, or disable its normal
device-attributes/cursor-position replies. It also does not by itself define
how replies or unwrapped payloads reach a custom control-mode renderer.

Do not confuse passthrough with these other mechanisms:

- `%output` and `%extended-output` expose pane output to control clients with
  octal escaping. Ttyphoon decodes it and queues it into its own terminal parser.
  tmux may still parse the same bytes and send a reply of its own.
- `pipe-pane` copies or injects pane data; it does not disable tmux's parser.
- `terminal-features` and `terminal-overrides` describe the outer terminal to
  tmux; they are not switches that delegate all application queries to it.
- Version-specific `refresh-client -r` reports can supply selected information
  from a control client, such as colour reports. They are not a universal raw
  query-forwarding interface.

### Response Ownership in Ttyphoon

Define the responder for each query before enabling a new capability. Existing
primary device-attributes code in `virtualterm/ansi_csi.go` suppresses ttyphoon's
reply when tmux is enabled, illustrating why adding a second responder is unsafe.

For new protocols, establish all of the following:

1. Requests reach the intended parser, including any passthrough envelope.
2. Replies reach the originating pane, not a different pane selected later.
3. There is one effective responder for each query, without conflicting or
   duplicate replies.
4. Ordering-sensitive probes work when requests share a single output chunk or
   are split across chunks. In particular, tmux must not answer the following
   device-attributes query before ttyphoon can answer a graphics probe.
5. Attach, restore, pane changes, alternate screens, and nested tmux do not
   accidentally advertise a capability that the active path cannot provide.

Existing output ordering alone does not establish query-response ordering:
tmux's replies can be generated before a control client processes the output.
Filtering what ttyphoon receives cannot retract a reply already written into
the application's input.

## Consequences

- Capability support can be implemented incrementally without impersonating
  another emulator or claiming unsupported protocol variants.
- Detection must be tested through ttyphoon's actual tmux/control-mode path,
  not only against the standalone ANSI parser.
- `allow-passthrough` is useful for cooperating applications but is not a
  blanket solution for replacing tmux's terminal-query responder.
- Making ttyphoon the sole responder for arbitrary existing applications may
  require a targeted upstream tmux change or a transport where ttyphoon owns the
  application PTY and tmux cannot answer first. Neither approach is selected or
  implemented by this record. Any intermediary would need to mediate both
  directions, preserve tmux state, and explicitly solve reply ownership.
- Verify behavior against the installed tmux version/build. Upstream image
  handling and available features are version/build dependent; the reviewed
  source contains image handling conditional on `ENABLE_IMAGES`.

## Verification

Future protocol work should use an isolated tmux server and a synthetic client
that captures actual application input. Test tmux alone versus ttyphoon-attached
behavior, bare versus DCS-wrapped requests, graphics query followed by primary
device attributes, multiple panes, supported/unsupported image modes, response
counts and order, and reconnect behavior. Do not change the user's running tmux
configuration to conduct these tests.

This ADR records source/manual research and the reporting policy. It does not
claim that the above protocol integration or suppression experiments have run.

## Reference

- [Environment setup](../config/env.go).
- [tmux feature setup](../tmux/session.go).
- [Control-mode output handling](../tmux/tmux.go).
- [Device-attributes replies](../virtualterm/ansi_csi.go).
- [Terminal reply path](../virtualterm/term.go).
- [Kitty graphics discovery](https://sw.kovidgoyal.net/kitty/graphics-protocol/#querying-support-and-available-transmission-mediums).
- [OSC 8 protocol](https://github.com/egmontkob/Hyperlinks/blob/master/README.md).
- [Ghostty terminfo guidance](https://ghostty.org/docs/help/terminfo).
- [tmux manual](https://github.com/tmux/tmux/blob/master/tmux.1), especially
  `allow-passthrough`, `terminal-features`, control mode, and `refresh-client`.
- [tmux query and passthrough handlers](https://github.com/tmux/tmux/blob/master/input.c).
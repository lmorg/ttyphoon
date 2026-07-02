# Vim-mode key coverage

## Implemented

### Mode entry
| Key | Action |
|-----|--------|
| `Esc` | Enter Normal mode |
| `i` | Insert before cursor |
| `a` | Insert after cursor |
| `I` | Insert at first non-blank of line |
| `A` | Insert at end of line |
| `o` | Open new line below, enter Insert |
| `O` | Open new line above, enter Insert |
| `R` | Enter Replace mode (overtype until Esc) |
| `r{char}` | Replace single char under cursor, stay Normal |

### Navigation (Normal mode)
| Key | Action |
|-----|--------|
| `h` / `←` | Left one character |
| `l` / `→` | Right one character |
| `j` / `↓` | Down one line |
| `k` / `↑` | Up one line |
| `w` | Forward to start of next word (punctuation-aware) |
| `b` | Back to start of previous word |
| `e` | Forward to end of word |
| `W` | Forward to start of next WORD (whitespace-delimited) |
| `B` | Back to start of previous WORD |
| `E` | Forward to end of WORD |
| `0` | Start of line (column 0) |
| `^` | First non-blank character of line |
| `$` / `End` | End of line |
| `gg` | First non-blank of file |
| `G` | Last non-blank of file |
| `{n}G` | Go to line n |
| `%` | Jump to matching bracket `( ) [ ] { }` |

### Operators (combine with motion, or double for whole line)
| Key | Action |
|-----|--------|
| `d{motion}` | Delete range |
| `dd` | Delete current line |
| `D` | Delete to end of line |
| `c{motion}` | Delete range and enter Insert |
| `cc` | Change current line |
| `C` | Change to end of line |
| `y{motion}` | Yank (copy) range into internal register |
| `yy` / `Y` | Yank current line |
| `x` | Delete character under cursor |
| `X` | Delete character before cursor |
| `p` | Paste yanked text after cursor / below line |
| `P` | Paste yanked text before cursor / above line |
| `u` | Undo (delegates to `document.execCommand('undo')`) |

All operators and motions accept a **count prefix** (e.g. `3w`, `2dd`, `5j`).

---

## Not yet implemented

### High value — straightforward
| Key(s) | Meaning |
|--------|---------|
| `f{char}` / `F{char}` | Find char forward/backward on current line |
| `t{char}` / `T{char}` | Till char forward/backward (stops one before) |
| `;` / `,` | Repeat last `f`/`t`/`F`/`T` forward/backward |
| `{` / `}` | Jump to previous/next empty line (paragraph boundary) |
| `J` | Join current line with the line below |
| `~` | Toggle case of char under cursor, advance |
| `.` | Repeat the last change |
| `s` | Substitute char under cursor, enter Insert (= `cl`) |
| `S` | Substitute whole line, enter Insert (= `cc`) |

### Medium value — more complex
| Key(s) | Meaning |
|--------|---------|
| `>>` / `<<` | Indent / unindent current line |
| `>` / `<` with motion | Indent / unindent range |
| `/` / `?` | Search forward/backward (requires mini search input) |
| `n` / `N` | Repeat last search next/previous |
| `*` / `#` | Search word under cursor forward/backward |

### Not practical in a textarea
| Key(s) | Reason |
|--------|--------|
| `H` `M` `L` | High/Middle/Low of screen — depends on visible viewport, not meaningful in a scrollable textarea |
| `Ctrl+D/U/F/B/E/Y` | Scroll commands — all Ctrl keys intentionally pass through to the host |
| `"` registers / `` ` `` `'` marks | Stateful, significant complexity |
| `q` macros | Requires macro recording state machine |

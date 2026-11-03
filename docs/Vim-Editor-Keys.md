# Vim-style Editor Keys

The Notes editor and Jupyter code blocks support Vim-style keyboard navigation. Press `Escape` to enter Normal mode; a mode indicator appears in the bottom-right corner of the editor. Press `i` (or any insert command) to return to regular editing.



## Mode entry

| Key | Action |
|-----|--------|
| `Esc` | Enter Normal mode |
| `i` | Insert before cursor |
| `a` | Insert after cursor |
| `I` | Insert at first non-blank of line |
| `A` | Insert at end of line |
| `o` | Open new line below, insert |
| `O` | Open new line above, insert |
| `R` | Enter Replace (overtype) mode — all typed characters overwrite |
| `r{char}` | Replace single character under cursor, stay in Normal |

## Navigation (Normal mode)

| Key | Action |
|-----|--------|
| `h` / `←` | Left |
| `l` / `→` | Right |
| `j` / `↓` | Down |
| `k` / `↑` | Up |
| `w` | Forward to start of next word |
| `b` | Back to start of previous word |
| `e` | Forward to end of word |
| `0` | Start of line (column 0) |
| `^` | First non-blank character of line |
| `$` / `End` | End of line |
| `gg` | First non-blank of file |
| `G` | Last non-blank of file |
| `{n}G` | Go to line *n* |
| `%` | Jump to matching bracket (`(` `)` `[` `]` `{` `}`) |

## Operators (combine with a motion or double for whole line)

| Key | Action |
|-----|--------|
| `d{motion}` | Delete range covered by motion |
| `dd` | Delete current line |
| `D` | Delete to end of line |
| `c{motion}` | Delete range and enter Insert |
| `cc` | Change current line |
| `C` | Change to end of line |
| `y{motion}` | Yank (copy) range into internal register |
| `yy` / `Y` | Yank current line |
| `x` | Delete character under cursor |
| `X` | Delete character before cursor |
| `p` | Paste yanked text after cursor |
| `P` | Paste yanked text before cursor |
| `u` | Undo |

All operators and motions accept a **count prefix** (e.g. `3w`, `5dd`, `10j`).

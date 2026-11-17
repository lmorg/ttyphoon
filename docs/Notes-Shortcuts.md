# Notes tab and split-view shortcuts

These shortcuts work when Notes has focus. Cmd is the Command key and Option
is the Alt key. Document-tab and split-view shortcuts do not run while a Notes
dialog or fullscreen image is open.

| Shortcut | Action |
|---|---|
| `Cmd+Shift+]` | Next document tab |
| `Cmd+Shift+[` | Previous document tab |
| `Cmd+W` | Close the active document tab |
| `Cmd+Shift+T` | Reopen the most recently closed document tab |
| `Cmd+\` | Split the focused view to the right |
| `Cmd+Option+Arrow` | Focus the neighbouring view in that direction |
| `Ctrl+Tab` / `Ctrl+Shift+Tab` | Next / previous Notes mode tab (View, Edit, etc.) |

If there is no neighbouring view in the requested direction, the focus shortcut
is left to the editor. Closing a dirty tab saves it first; if saving fails, the
tab stays open.

The document tab context menu also offers **Close tab**, **Split right**, and
**Split down**. Middle-click closes a tab. The command palette offers the tab
navigation, close, reopen and split commands, including Split down.

Dragged document tabs have four rounded corners, a border on every edge and
75% opacity; tabs in the tab bar keep their usual style.

Drag a document tab onto a pane's right or bottom edge to move it into a new
split. The outer quarter of the pane activates an accent-coloured preview of
the new half-pane; dropping there creates the split and removes the tab from
its original tab bar. An emptied source pane closes. Dragging a pane's only tab
onto its own edge does nothing. Dropping on a tab bar still reorders or moves
the tab without splitting.

For editor Vim keys, see [Vim-Editor-Keys.md](Vim-Editor-Keys.md). Configurable
application hotkey defaults are listed in [defaults.yaml](../config/defaults.yaml);
this page is not a complete reference for every application shortcut.

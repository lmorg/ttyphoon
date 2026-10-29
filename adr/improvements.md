# Improvements

All improvements should consider if actions are blocking the UI -- particularly when loading in files externally where we are unsure of their file size. If so, we should consider lazy loading and using a CSS spinner to demonstrate that view is being loaded.

## features

- [x] AI command tools: consolidate `interactive` and `bash` into one clone-backed Bash script tool ([ADR 0056](../adr/0056-shell-tools-separate-tty-and-script-execution.md))
- [x] Notes, Git: add tracked-only Stage all and a collapsible commit row with commit-message input and output handling ([ADR 0062](../adr/0062-notes-git-worktree-panel.md))
- [x] Notes, graphs: copy graph as PNG, modify an existing graph from graph/table context menus, and retain Remove graph after a separator ([ADR 0054](../adr/0054-persisted-graphs-for-notes-tables.md))
- [ ] Notes, Hex mode: should support editing and saving
- [x] Notes: History stores line numbers, jumps to the saved line when navigating via the toolbar arrows, and records same-document positions at least 20 lines apart
- [ ] AI panel: export output to Markdown (ie in case you forget to tick the save output checkbox when submitting the prompt)
- [ ] Spellcheck: user dictionaries:
  - [ ] global (across every document in every project)
  - [ ] language (similar to reserved words, so language specific but applied across every project)
  - [ ] workspace (specific to workspace name, ie terminal tab name)
- [x] Terminal, notes and tools tabs should have --accent hover colour
- [x] Terminal: maximize button. This should reuse the same maximize model that Notes uses when maximized
- [ ] File list: support to move files by drag and dropping
- [x] File list: right click on a folder: menu option `Create new file...`. This will open the create file model with the path to that folder pre-populated
  - [x] File list: right click on a folder: menu option `Paste as new file...`. This will copy the contents of clipboard to memory, then show the create file model with the path prepopulated and the file name either `clipboard.png` (for images) or `clipboard.md` (for text). Clipboard access uses existing Go bindings rather than web APIs.
- [x] Terminal: hotkey for tmux pane zoom
- [x] AI, read file tool: should support reading chunks defined by line number ranges so that large files don't cause us to run out of tokens
- [ ] AI, skills: improve support for idiomatic skills files
- [x] Notes, CSV: in `View` mode, the markdown auto-URL tool should be invoked (inc the support for custom url regexp)
- [x] Notes: cache the sizing of the `file list | notes editor` splitter / resizer ([ADR 0059](../adr/0059-notes-layout-sizing-persistence.md))
- [x] Notes: cache the sizing of the `notes | terminal` splitter / resizer ([ADR 0059](../adr/0059-notes-layout-sizing-persistence.md))
- [x] Notes: add support for the tools panel to be resized ([ADR 0059](../adr/0059-notes-layout-sizing-persistence.md))
  - [x] Notes: cache and restore the resized tools-panel dimensions
- [x] Terminal: the gaugebars should have the same styling as the scrollbars in Notes: width, default colour, rounded corners, alpha transparency
- [x] AI: once the agent has completed, if autoscrolling is enabled (eg we don't see [Latest] button), the AI panel scrolls to the start of the final returned answer, not tool calls or thinking ([ADR 0035](../adr/0035-ai-output-explicit-follow-mode.md))
- [x] Chore: consolidate all the references of ~/Documents/ttyphoon into one function in the `file` package ([ADR 0065](../adr/0065-shared-documents-directory.md))
- [x] When the git panel is open, add a watch for .git directory to push updates to the git tab.
- [x] Terminal: tmux window / pane resizes should push the pane dimensions to the frontend ([ADR 0072](../adr/0072-interactive-pane-actions-use-one-shot-uuid-callbacks.md)).
  - [x] Show a close button at the top right of completed panes retained for inspection, including ClonePane actions.
  - [x] Successful ClonePane actions close after 30 seconds with a notification-style countdown; click the timer to cancel auto-close and retain the pane for manual closing.
  - [ ] Using this new data, the frontend can then support a tmux zoom for the pane by placing the maximize button in the top right corner of the pane rather than in the terminal toolbar
- [ ] Notes: support for displaying video files
- [ ] Notes: support for audio playback
- [ ] Notes, Git commits log: a toggle below the filter search box to choose whether to search for commit messages or file names. Default is commit messages. When file search is toggled then each commit is checked to see which files were changed as part of that commit, with those filenames being what's filtered
- [x] AI tool, graphs: this currently renders SVG. It should do PNG instead
- [x] AI tool: read-only Git inspection for status, diff, log, show, branches, and remotes; enabled and always allowed by default, run through ClonePane ([ADR 0082](../adr/0082-ai-read-only-git-tool.md))
- [x] AI: per-prompt Effort selector guides investigation depth and when to ask for blocking user feedback ([ADR 0081](../adr/0081-ai-prompt-effort-and-feedback-cadence.md))
- [x] Notes: spellcheck suggestions open on right-clicking a misspelt word; left-click remains for editing ([ADR 0069](../adr/0069-notes-spellcheck-suggestions-open-on-right-click.md))
- [ ] Our custom menu should support ctrl+v / cmd+v to paste text to the usually hidden search input field
- [ ] Notes, git panel: hide and disable the commit toolbar button when there are no changes to commit and nothing to stage (clean work tree)
- [ ] Terminal: store the initial state env vars in case we don't receive the env vars via a ANSI escape block. Those initial vars should be what's passed to tmux during the split tmux comment, or the app env vars if the formmer is also unknown
- [x] Notes, `Edit` mode: the autosave should also update the ToC and Frontmatter content in the Tools panel
- [x] Notes, Inputbox: add support for pasting images (this should be an flag added to the inputbox struct to enable. It should be disabled by default).
  - [x] AI: images should be uploaded to the LLM as part of the query
- [x] Notes, Inputbox: add support to "Save draft". This will close the dialog but cache that the values for all fields (including additional fields referred to as "variables"). Then the next time that inputbox with the same parameters is opened, pre-populate all those fields
- [x] Notes, Inputbox: any additional fields (variables) are not stored in the history like the main input field is. I'd like all fields to be stored in history. Clicking the [...] should update all fields. But the up arrow on text fields should return history for just that field
- [ ] Notes, SVG: these files should support an Edit even though they're images because they are built from text
- [ ] Terminal: kitty graphics support (this should be a new ADR)
- [x] Git panel: branch fetch, tag fetch, and tag push use ClonePane so remote credentials can be entered interactively ([ADR 0071](../adr/0071-git-package-owns-action-execution-policy.md))
- [x] Git panel: Push button next to Commit (icon `0xf093`) asks for confirmation, then runs the configured `git push` through ClonePane ([ADR 0062](../adr/0062-notes-git-worktree-panel.md), [ADR 0071](../adr/0071-git-package-owns-action-execution-policy.md))
- [x] Git panel: keep Commit in the collapsible row and place the icon-style Push button in the main toolbar beside the commit toggle ([ADR 0062](../adr/0062-notes-git-worktree-panel.md))
- [x] Kill pane confirmation modal: Tab/Shift+Tab switches between buttons; Enter/Space activates the focused button; text and terminal shortcuts are ignored ([ADR 0072](../adr/0072-interactive-pane-actions-use-one-shot-uuid-callbacks.md))
- [x] AI, tool requests: Deny opens the shared inputbox for an optional free-form reason; Cancel/Draft leave the request pending, submitted reasons reach the model and are remembered only within the tmux workspace/prompt scopes ([ADR 0002](../adr/0002-tool-permission-scopes.md))
- [ ] terminal: hover over buttons (eg maximize, close, etc) should have translucent backgrounds (0.5 alpha)

## refactors

- [ ] Notes: move LSP code into new JS file

## functional tweaks

- [ ] Window UI: disable the custom titlebar on non-macOS platforms

## reimplement

- [x] AI: services should follow a new hierarchy:
  ```
  - Service: "string"       # this can be any title the user chooses
    - description: "string" # user defined description. we can use this in things like the AI settings
    - Provider: "string"    # this will be eg Anthropic, OpenAI, Ollama, etc
    - Models: [ "claude-opus-8", "etc" ] # this will be the models available for that service
    - Default: "string"     # name of modeel to default to for that service
    - Env { "key": "value" } # optional list of environmental variable key value pairs for that service. eg API key, Ollama host url, etc
  ```
- [x] AI, tool: existing file writer should be for **new** files. A new tool should be created for editing existing files. This should support patching instead

## bugfixes

- [x] Notes, AI panel: prompt for a name when creating a session; blank names retain the existing `New session` default ([ADR 0027](../adr/0027-shared-single-line-text-prompt.md))
- [x] Notes, Git: classify fetch/tag transfer output by command outcome; successful stderr is info in Log, with one completion notification, while failed commands show an error ([ADR 0062](../adr/0062-notes-git-worktree-panel.md))
- [x] Notes, AI/Log fullscreen: hide the entire Tools tab-bar header rather than individual tabs, including Git and Frontmatter ([ADR 0059](../adr/0059-notes-layout-sizing-persistence.md))
- [x] Terminal: leaving the terminal canvas clears hover chrome and element hover state, including when moving into Notes ([ADR 0067](../adr/0067-terminal-hover-is-scoped-to-pointer-presence.md))
- [x] Terminal: End moves to the end of the line with tmux enabled; the browser key name now reaches the existing terminal encoding ([ADR 0066](../adr/0066-end-key-name-must-reach-terminal-encoding.md))
- [x] Notes, Find panel: JSON/YAML/XML Edit mode searches and selects matches in the live Monaco editor ([ADR 0021](../adr/0021-editor-navigation-must-target-the-live-surface.md))
- [x] Notes, Find panel: Cmd+F / Ctrl+F opens the Notes Find panel before Monaco handles its own search shortcut ([ADR 0021](../adr/0021-editor-navigation-must-target-the-live-surface.md))
- [x] Notes: rename causes a `Files does not exist` error if that file is open. Notes should reopen it with the new file name ([ADR 0064](../adr/0064-notes-file-mutation-navigation.md))
- [ ] Terminal: npm test causes flickering. Why?
- [ ] AI: Responses, menu options, all use `Service()` but I think `Model()` is more apt
- [ ] Languages: closing tags pairs should support overtyping. eg typing `(` auto completes with `)` but if you type `(` followed by `)` then you end up with `())`. Ideally the last `)` should overwrite the existing `)`
- [x] Notes: switching terminal tabs doesn't cause file list to be refreshed
- [x] Notes, file list: this should be narrow default % width
- [x] Notes: deleting a file switches to the previous open document if that file is presently open in Notes, restoring its saved line ([ADR 0064](../adr/0064-notes-file-mutation-navigation.md))
- [ ] Notes: when AI opens a document, that document should be added to document history for that workspace
- [x] Notes, Find panel: clicking the grep / "files containing" search mode buttons (eg `_W` for whole word) re-invokes the grep search (already implemented; verified for case, regex and whole word)
- [x] Notes, Find panel: searching doesn't jump to the position in JSON files
- [x] Notes, grep file list: when a document is saved, when a document is auto-saved: the search cache should be invalidated, a new search performed, and those results displayed in the file list, with the scroll position preserved
- [x] Notes, grep search for files containing: typing a new search query or clicking the search types (eg whole word, regex, case sensitive) performs a new search (query and options are part of the search signature).
- [x] Notes, monaco: when pasting an image into a markdown document, monaco editor isn't updated/refreshed automatically with the new contents of the markdown file
- [x] Notes, manaco: wordwrap should be enabled by default for markdown documents
- [ ] Window UI, titlebar: in macOS with the custom titlebar, sometimes resizing, dragging, double click to maximize and similar actions case the window to become highlighted as if you've dragged the caret to highlight text. This shouldn't happen
- [x] Notes: error when creating new document doesn't log actually error
- [x] Notes, monaco: ctrl+v don't invoke custom paste function (eg like when pasting from right-click context menu). which means images are not pasted
- [x] Notes: image pasting doesn't work if markdown document file name has a space in it. eg `hello world.md` will not paste/save/show the image but `hello-world.md` would.
- [x] Notes CSV: resizing columns doesn't work
- [x] AI: Unable to render description: template: write-file-auth-description:3:20: executing "write-file-auth-description" at <$v>: wrong type for value; expected string; got txtar.File
- [x] AI: bash errors cause the agent to fail: `2026/10/07 12:36:55 [error] [NodeRunError] failed to stream tool call call_QodbP47uKwMvrvU2lOSEjI1C: bash exited unsuccessfully: exit status 1 | ------------------------ | node path: [tools]`

## future ambitions

- [ ] Migrate to Wails 3
  - [ ] Notes: Printing should cause the view to go fullscreen so superfluous widgets aren't visible in the print
- [x] Git support: new tab in tools panel (between `Find` and `AI`) called `Git`. Selecting that should switch the file list to a git log tree which shows branches, commits and merge history. The `Git` panel should have two headings: `Status` and `Diff`, each which show the respective output from `git` but formatted for readability and colour in a Wails web frontend. Colours should follow the theme colours defined in CSS
- [ ] Notes: support having multiple documents open at a time
- [ ] Extensibility: plugin support (no idea how this would be implemented, what features would be available to plugins, etc)
- [ ] Notes, image: support for annotating images. Add text, highlights, ovals and rounded rectangles, with an option to export the annotations flattened onto a PNG while preserving the original image.
- [ ] AI: run agents inside a docker container. $GLOBAL, $PROJECT and $NOTES should all be mounted directories. Tools and MCP should still work but inside the docker container.

# Completed

## features

- [x] Notes: Vim mode for text editors
- [x] Notes, file filter: exclusion filter should use the Go `find` package: /utils/find -- JS should send the list of files as an array to frontend.go and frontend.go should call Filter()
- [x] Notes: spellchecker. Ideally result the existing spellcheck package in Go. And the same -red curly underline style used in the terminal
- [x] Notes, JSON/YAML View mode: documents are slow to load, we should stream HTML elements instead of rendering them all at the start (performance improvement)
- [x] Notes: Mouse hovering over a hyperlink should display the URL in a popup tooltip (like with LSP). Moving the cursor away from the link should automatically hide the tooltip. Don't set use the OS native tooltip
- [x] Notes: View HTML files. Ideally in an iframe with scripts disabled
- [x] Find panel: Find, replace, and grep search terms should be cached (90 days)
- [x] Notes: completely rework the undo/redo engine to be bespoke rather than relying on webkit. ensure that the undo history is cached (10 days) per document so you can switch documents and resume
- [x] Notes, JSON/YAML View mode: collapse/expand all nodes 
- [x] Notes, markdown: support for frontmatter. Should be a tab in tools (next tab after ToC) called `Frontmatter` and rendered using the JSON/YAML renderer. The markdown render in `View` and `Run` mode should have a caption that says:
  ```markdown
  This document contains Frontmatter. [View]
  ```
  where `View` is a button that opens the Tools panel and switches to the Frontmatter tab.
- [x] Notes, JSON/YAML view mode: support adding new keys, and deleting keys. This should be available via right clicking. It should be supported by both frontmatter and the JSON/YAML view mode
- [x] Terminal: hover over tooltip effect. This will need to be a new method in the renderer Go and JS. The hyperlink element will call that renderer method.
  - [x] Notes: update notes to call this new renderer method

## refactors
- [x] Notes: custom regex hyperlinks should call the same JS function as regular hyperlinks and thus consolidate the left/middle/right click logic
- [x] Notes: move CSS rules into a `.css` file

## functional tweaks
- [x] Notes, Tools panel: preserve the selected tab across document, project-root and workspace changes; fall back to Find only when that tab is unavailable
- [x] Notes: disable select to copy to clipboard
- [x] Notes: headings should collapse/expand on double click, not single click
- [x] Notes: table headings horizontal alignment should should be left aligned not center aligned
- [x] Notes: middle click should not paste from clipboard

## reeimplement

- [x] Terminal: rewrite text reflow logic when terminal is resized
- [x] Notes, JSON/YAML View mode: When editing keys/values in View mode, the same should only edit that key/value and not write the entire file. Overwriting the entire file causes changes to formatting which causes problems for git diffs and loss of comments in YAML
- [x] Notes: LSP: remove the highlight that appears over every single keyword. This is already automatically disabled on large documents. I think it should be permanently disabled

## bug fixes

- [x] Notes: Edit editor view doesn't horizontally scroll with the caret when jumping with navigation keys such as home/end. Meaning text isn't visible when typing off screen
- [x] Notes: Find in document scrolls vertically when next/previous through matches, but the view doesn't scroll horizontally
- [x] Notes: Tools button goes transparent when hover. The transparency shouldn't change
- [x] Notes: Tools panel raise/lower state shouldn't be cached
  - [x] Notes: Should Tools panel tab be cached?
- [x] Notes: maximize button bg should be --accent and fg should be --bg
- [x] Inputbox model: numeric (integer) variables have controls for increasing and decreasing the number. Those controls are black and hard to see. So the controls should have a colour of --fg and a background of --bg
- [x] Languages, Markdown: tab index should be 2
- [x] Notes: when fullscreen, the Edit --accent border should have a border radius of 8px for ONLY the bottom right corner, so the --accent highlight curves around the fullscreen container element's border radius
- [x] AI: skills should be return in an alphabetically sorted order
- [x] Notes: when editor is fullscreen and scrolled horizontally, restoring the editor into normal view causes text to be cropped (scrolled fully to left but rendering the text from the point where it was previously scrolled). manually scrolling again fixes the issue
- [x] Notes: word wrap for tables is toggled via a right click menu option. By default it is disabled. I'd like it to be enabled by default

## future ambitions

- [x] Migrate to Eino
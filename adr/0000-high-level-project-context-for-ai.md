# Project Context for LLMs

This is a terminal emulator and ide.
- The terminal part of the project is called `terminal`
    - The terminal supports multiplexing via `tmux`
    - Each terminal tab is a tmux `window`. We'll refer to this as a `workspace`
    - The terminal/tmux pane's working directory is used to derive the workspace's top level project directory (typically the by scanning upwards toward root for `.git`)
- The IDE part is called `Notes`
    - Notes supports multiple different modes depending on the document type, such as jupyter, swagger, markdown, hex viewer, image viewer and so on
    - Notes projects are scoped by 
- There is a sqlite3 database that is used for caching
- Changes are tracked in ADR (architecture decision records) in `/adr`.
    - Use the ADRs to gain context about specific parts of application before making changes
    - Any changes you make should also update the ADRs
- The architecture is broadly split into two distinct codebases: backend and frontend
    - The backend is written in Go lang. This is where the virtualterm logic etc will live.
    - The frontend uses web technologies and called via OS native webviews such webkit on macOS
    - The frontend and backend commincate via an IPC using the Go package `wails`
    - In general, most of the core logic should live in the backend except for when that logic exclusively relates to the frontend UI/UX.
- Any changes can be tested using `npm` and/or `go test`.
- The application has an inbuilt agent harness that is scoped per workspace

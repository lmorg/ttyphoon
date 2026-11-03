- Open local files for reading and return their contents.
- Bare filenames and relative paths use the owning terminal pane's current project root, resolved dynamically. Absolute paths must be inside that root.
- Useful for debugging output that references local files.
- The output of this tool conforms to the `txtar` specification.
- Files that cannot be opened are returned with contents saying `!!! Cannot open file`.
- Input is a JSON object with a `files` array of objects, each containing `path`
  and optional `startLine`/`endLine`. Pass it directly as tool arguments; do not
  wrap it in an `input` property. Bounds belong to each file, never at the top level.
- Each file independently selects a 1-based inclusive range. Omit its bounds
  to read that entire file. With only `endLine`, start at line 1;
  with only `startLine`, read through EOF. Bounds must be positive and the end
  must not precede the start. Beyond EOF, return only available lines (or empty
  content when the start is beyond EOF). Mix different ranges and whole-file
  reads in one call. Legacy string entries are accepted as whole-file reads.
- Ranged reads stream only the needed prefix, without loading the whole file.

```json
{"files":[{"path":"src/main.go"},{"path":"README.md"}]}
```

Read lines 5 through 20 of one file, a different range of another, and a whole file:

```json
{"files":[{"path":"src/main.go","startLine":5,"endLine":20},{"path":"src/helper.go","startLine":30,"endLine":40},{"path":"README.md"}]}
```
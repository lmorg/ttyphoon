- Open local files for reading and return their contents.
- Useful for debugging output that references local files.
- The output of this tool conforms to the `txtar` specification.
- Files that cannot be opened are returned with contents saying `!!! Cannot open file`.
- Input is a JSON object with a `files` array of paths. Pass it directly as tool
  arguments; do not wrap it in an `input` property.

```json
{"files":["src/main.go","README.md"]}
```
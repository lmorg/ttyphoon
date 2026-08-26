Patches an **existing** file by replacing exact blocks of text.

Use this tool for every edit to a file that already exists. Use `writeFile`
only when creating a brand new file.

Patching is preferred over rewriting because it keeps the parts of the file you
are not changing untouched, it avoids spending tokens re-emitting the whole
file, and it produces clean diffs.

The input for this tool MUST be a JSON object:

```json
{
  "file": "path/to/file.go",
  "edits": [
    {
      "old": "exact text to find, including indentation and newlines",
      "new": "replacement text"
    }
  ]
}
```

Rules:

- `file` is relative to the working directory, or an absolute path within it.
  The file must already exist.
- `old` MUST match the file contents **exactly**, byte for byte, including
  whitespace, indentation and line breaks. Do not add ellipses or placeholder
  comments.
- `old` MUST match exactly one location in the file. Include several
  surrounding lines of unchanged context to make the match unique. If it
  matches zero times or more than once the whole patch is rejected and the file
  is left unchanged.
- To delete text, set `new` to an empty string.
- To insert text, set `old` to a unique anchor and set `new` to that same
  anchor plus your new lines.
- `edits` are applied in order, and each is matched against the result of the
  previous ones. Supply multiple edits in a single call rather than calling the
  tool repeatedly.

Read the file first if you are not certain of its exact current contents.

On success the tool reports how many edits were applied. On failure it reports
which edit failed and why, and no changes are written to disk.

Inserts new lines into an **existing** file at given line numbers.

Use this tool when you are adding whole lines and the surrounding text is
unchanged, for example appending an entry to a list, adding an import, or
appending to the end of a file. Use `patchFile` when you need to replace or
delete existing text, and `writeFile` only when creating a brand new file.

The input for this tool MUST be a JSON object:

```json
{
  "file": "path/to/file.go",
  "inserts": [
    {
      "line": 12,
      "text": "\tnewLine := true\n\tanotherLine := false"
    }
  ]
}
```

Rules:

- `file` is relative to the working directory, or an absolute path within it.
  The file must already exist.
- `line` is the 1-indexed line number to insert **after**. Use `0` to insert at
  the very start of the file. Use the file's last line number to append to the
  end.
- `text` may contain multiple lines separated by `\n`. Include the exact
  indentation you want; nothing is added or adjusted for you.
- All `line` numbers refer to the file **as it is now**, before any of the
  inserts in this call are applied. You do not need to adjust later line
  numbers to account for earlier inserts.
- Supply multiple inserts in a single call rather than calling the tool
  repeatedly.

Read the file first so your line numbers are correct.

On success the tool reports how many inserts were applied. If any `line` is out
of range the whole call is rejected and no changes are written to disk.

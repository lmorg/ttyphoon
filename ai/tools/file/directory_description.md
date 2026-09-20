- Output matching files and directories under the current project root.
- Returns a JSON array of objects: `[{"Name":"relative/path","IsDir":false}]`.
- `Name` is relative to the project root. `IsDir` is true for directories.
- Input is a filter, not a path. Use it to include or exclude specific files
	or directories. Use an empty string to list everything.

## Filter rules

- default: match all space-separated words, eg `readme md` matches only paths
	containing BOTH `readme` and `md`
- contains `*` or `?`: glob match, eg `*.go` for Go files
- `g ` prefix: glob match, eg `g *.go` for Go files
- `rx ` prefix: regex match, eg `rx .*\.go$` for Go files
- `or ` prefix: match any space-separated word, eg `or md txt` for Markdown
	or text files
- `! ` prefix: reject paths that match all space-separated words, eg `! md txt`
	excludes paths containing BOTH `md` and `txt`

All filter matches are case insensitive.
All filter rules apply to directories too. Configured excluded directories are
skipped while walking.

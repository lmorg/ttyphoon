Search project files for text containing a query phrase. Returns matching file
names, absolute paths, line numbers, and one line of context before and after
each match. Results are paged at up to 50 matches per page.

Input:
```json
{
	"query": "search string",   # required for a new search; omit only when requesting another page from the previous search
	"options": {
		"caseSensitive": false, # true = return case sensitive matches only
		"regex": false,         # true = query string is a regex pattern
		"wholeWord": false,     # true = query string will not match substrings of words 
		"fileFilter": "",       # filter which files to query. Or omit to scan all files in project
	},
	"page": 2 # page number to return from the previous search. Omit for a new search. Page numbers start at 1
}
```

fileFilter syntax:
- default: match all space-separated words, eg `readme md` matches only paths
  containing BOTH `readme` and `md`
- contains `*` or `?`: glob match, eg `*.go` for Go files
- `g ` prefix: glob match, eg `g *.go` for Go files
- `rx ` prefix: regex match, eg `rx .*\.go$` for Go files
- `or ` prefix: match any space-separated word, eg `or md txt` for Markdown
  or text files
- `! ` prefix: reject paths that match all space-separated words, eg `! md txt`
  excludes paths containing BOTH `md` and `txt`

All fileFilter matches are case insensitive.

Returns:
```json
{
	"error": "error message, if applicable",
	"pageCount": 5,  # number of pages of results. Each return is a single page
	"pageNumber": 1, # the index of this page in the page count. Indexes start from 1
	"results": [ # max 50 items per page
		{
			"fileName": "example.md",
			"path": "/absolute/path/to/project/example.md",
			"line": 15,
			"context": [
				"line before matched line",
				"this is the matched line",
				"line after matched line"
			]
		}
	]
}
```

If `pageCount` is greater than 1, the return contains partial results. Request
the next page with `{"page": 2}`, and continue until `pageNumber` matches
`pageCount`. Multiple pages can consume a lot of tokens, so prefer refining the
query or `fileFilter` when possible.

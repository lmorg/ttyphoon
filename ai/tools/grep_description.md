Searches project files for text containing a query phrase and returns a list of file names, line numbers and the lines preceding and following the matched query. Supports additional filters.
Input:
```json
{
	"query": "search string",   # omit if requesting a new page from the previous search 
	"options": {
		"caseSensitive": false, # true = return case sensitive matches only
		"regex": false,         # true = query string is a regex pattern
		"wholeWord": false,     # true = query string will not match substrings of words 
		"fileFilter": "",       # filter which files to query. Or omit to scan all files in project
	},
	"page": 0 # which page of results to return. Omit this if new search. Page count is included in return
}
```

fileFilter syntax:
- default: must match all of the space-separated words, eg "readme md" for only files with BOTH "readme" and "md" in the filename or extension
- contains "*" or "?": glob match, eg "*.go" for files with "go" extensions
- "g " prefix: glob match, eg "g *.go" for files with "go" extensions
- "rx " prefix: files must match a regex pattern, eg "rx .*\.go$" for files with "go" extensions
- "or " prefix: files much match any of the space-separated words, eg "or md txt" for any files containing the word "md" or "txt" in the file name or extension
- "! " prefix: files much not match all of the space-separated words, eg "! md txt" for any files that DO NOT contain either the word "md" nor "txt" in the file name or extension
All fileFilter matches are case insensitive

Returns:
```json
{
	"error": "error message, if applicable",
	"pageCount": 5,  # number of pages of results. Each return is a single page
	"pageNumber": 1, # the index of this page in the page count. Indexes start from 1
	"results": [ # max 50 items per page
		"fileName": "example.md",
		"path": "path/in/project/example.md",
		"line": 15,
		"Context": [
			"line before matched line",
			"this is the matched line",
			"line after matched line"
		]
	]
}
```
If pageCount is greater than 1, then the return only contains partial results and you'll need to make another request with {"page":2} to get the next page...and so on until pageNumber matches pageCount. However if pageCount is > 1 then that might be an indicator that a refined search is required. Multiple pages can result in running out of tokens

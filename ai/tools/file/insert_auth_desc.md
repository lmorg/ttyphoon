File: `{{ escapeMarkdown $.File }}`
| # | Text |
| --- | --- |{{ range $v := .Inserts }}
| {{ add $v.Line 1}} | {{ markdownTableCode $v.Text }} |{{ end }}


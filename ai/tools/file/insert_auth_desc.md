File: `{{ escapeMarkdown $.File }}`
| # | Text |
| --- | --- |{{ range $v := .Inserts }}
| {{ add $v.Line 1}} | {{ escapeMarkdown $v.Text }} |{{ end }}


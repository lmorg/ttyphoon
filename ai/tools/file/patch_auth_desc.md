File: `{{ escapeMarkdown $.File }}`
| Change | Text |
| --- | --- |{{ range $v := .Edits }}
| `-` | {{ escapeMarkdown $v.Old }} |
| `+` | {{ escapeMarkdown $v.New }} |{{ end }}


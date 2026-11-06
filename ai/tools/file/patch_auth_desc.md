File: `{{ escapeMarkdown $.File }}`
| Change | Text |
| --- | --- |{{ range $v := .Edits }}
| `-` | {{ markdownTableCode $v.Old }} |
| `+` | {{ markdownTableCode $v.New }} |{{ end }}


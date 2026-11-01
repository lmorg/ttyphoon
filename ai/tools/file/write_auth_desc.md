| File path | Contents |
| --- | --- |{{ range $v := .Files }}
| {{ escapeMarkdown $v.Name }}{{ if $v.CreatesDirectories }}<br/>(creating directories too){{ end }}{{ if $v.PathError }}<br/>{{ escapeMarkdown $v.PathError }}{{ end }} | {{ markdownTableCode $v.Content }} |{{ end }}


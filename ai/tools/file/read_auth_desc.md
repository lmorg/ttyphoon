| File name | Lines (inclusive) |
| --- | --- |{{ range $v := .Files }}
| {{ escapeMarkdown $v.Path }} | {{ if or $v.StartLine $v.EndLine }}{{ if $v.StartLine }}{{ $v.StartLine }}{{ else }}1{{ end }} through {{ if $v.EndLine }}{{ $v.EndLine }}{{ else }}EOF{{ end }}{{ else }}Entire file{{ end }} |{{ end }}


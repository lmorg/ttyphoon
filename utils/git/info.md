# Info

`{{ .CommitHashLong }}`
{{ if .BranchName }}
Branch: `{{ .BranchName }}`
{{ end }}
{{ if .MatchedTags }}## Tags
{{- range $tag := .MatchedTags }}
- `{{ $tag }}`
{{- end }}{{ end }}
{{ if .IsWorktree }}
## Worktree Path

`{{ .RepoWorktreePath }}`

## Status

```
{{ .RepoStatus }}
```
{{ end }}
## Diff

{{- if .IsWorktree }}
Uncommitted changes:
{{- else }}
Changes in commit:
{{- end }}
| diff |
| --- |
{{- range $f := .RepoDiffFiles }}
| <details><summary>{{ escapeMarkdown $f.Filename }}</summary>{{ markdownTableCode $f.Diff }}</details> |
{{- end }}

## Git Executable

- Version:  `{{ .GitExeVersion }}`
- Path: `{{ .GitExePath }}`


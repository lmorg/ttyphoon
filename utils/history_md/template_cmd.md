# {{ .AppName }} History

## Meta

- Group: {{ .GroupName }}
- TilePane: {{ .TileName }}
- Timestamps:
  - Start: {{ .TimeStart }}
  - End: {{ .TimeEnd }}
  - Duration: {{ .TimeDuration }}
- Working directory: {{ .Pwd }}
- Host: {{ .Host }}
- Exit Code: `{{ .ExitNum }}`

## Command Line

```sh
{{ .Query }}
```

## Output

```{{ .OutputLang }}
{{ .Output }}
```

EOF
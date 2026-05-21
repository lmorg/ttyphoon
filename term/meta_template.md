# Command Metadata

## Row

> {{ quote (trim .RowString) }}

- Host: `{{ .Source.Host }}`
- Pwd: `{{ .Source.Pwd }}`

## Block

- Exit Code: `{{ .Block.ExitNum }}`
- Time Start: {{ .Block.TimeStart }}
- Time End: {{ .Block.TimeEnd }}

### Query

```murex
{{ toString .Block.Query }}
```

### Output

<details>
<summary>
Click to expand
</summary>

```text
{{ .Output }}
```

</details>

## Agent
{{ if .Block.AiMeta }}
- Agent: {{ .Block.AiMeta.Agent }}

### Prompt

> {{ quote .Block.AiMeta.Prompt }}

### Response 

> {{ quote .Block.AiMeta.Response }}
{{ else }}
_nil_
{{ end }}

### {{ .AppName }} Debug

- Block ID: `{{ .Block.Id }}`
- Meta ID: `{{ .Block.Meta }}`
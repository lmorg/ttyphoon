# Debug

> {{ .RowString }}

## Row

- ID: {{ .RowId }}
- Meta: {{ .RowMeta }}
- Source:
  - Host: {{ .Source.Host }}
  - Pwd: {{ .Source.Pwd }}

## Block

- ID: {{ .Block.Id }}
- Meta: {{ .Block.Meta }}
- Query: `{{ toString .Block.Query }}`
- Exit Code: {{ .Block.ExitNum }}
- Time Start: {{ .Block.TimeStart }}
- Time End: {{ .Block.TimeEnd }}

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
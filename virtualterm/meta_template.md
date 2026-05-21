# Metadata

## Environment

- Host: `{{ .Source.Host }}`
- Pwd: `{{ .Source.Pwd }}`

## Timings

| Property | Value | Units |
| --- | --- | --- |
| Date | {{ .DateStart }} | yyyy-mm-dd |
| Start | {{ .TimeStart }} | hh:mm:ss |
| End | {{ .TimeEnd }} | hh:mm:ss |
| Duration | {{ .Duration}} | milliseconds |

## {{ if .Block.AiMeta }}Query{{ else }}Command Line{{ end }}

```raw
{{ toString .Block.Query }}
```

## Return

Exit Code: `{{ .Block.ExitNum }}`

```text
{{ .Output }}
```

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

## {{ .AppName }} Debug

- Block ID: `{{ .Block.Id }}`
- Meta ID: `{{ .Block.Meta }}`
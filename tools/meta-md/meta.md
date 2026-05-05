# {{ .Filename }}

```text
{{ .PathFull }}
```

> {{ .FileType }}

## Attributes

- Size:
  - Human: `{{ .SizeHuman }}`
  - Bytes: `{{ .SizeBytes }}`
- Date:
  - Created:  `{{ .DateCreated }}`
  - Modified: `{{ .DateModified }}`

## Owners

- User:  `{{ .UserOwner }}`
- Group: `{{ .GroupOwner }}`

## Permissions

- Unix:  `{{ .UnixOctal }}`
- User:  `{{ .UserACL }}`
- Group: `{{ .GroupACL }}`
- Other: `{{ .OtherACL }}`
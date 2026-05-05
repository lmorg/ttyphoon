# {{ .Filename }}

> {{ .FileType }}

## Attributes

- Size:
  - Human: `{{ .SizeHuman }}`
  - Bytes: `{{ .SizeBytes }}`
- Path:
  ```text
  {{ .PathFull }}
  ```

## Owners

- User:  `{{ .UserOwner }}`
- Group: `{{ .GroupOwner }}`

## Permissions

- Unix:  `{{ .UnixOctal }}`
- User:  `{{ .UserACL }}`
- Group: `{{ .GroupACL }}`
- Other: `{{ .OtherACL }}`
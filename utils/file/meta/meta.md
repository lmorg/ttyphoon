# {{ .Filename }}

> {{ .FileType }}

## Attributes

- Size:
  - Human: `{{ .SizeHuman }}`
  - Bytes: `{{ .SizeBytes }}`
- Date:
  - Created:  `{{ .DateCreated }}`
  - Modified: `{{ .DateModified }}`
- Path:
  ```text
  {{ .PathOnly }}
  ```


## Owners

- User:  `{{ .UserOwner }}`
- Group: `{{ .GroupOwner }}`

## Permissions

- Unix:  `{{ .UnixOctal }}`
- User:  `{{ .UserACL }}`
- Group: `{{ .GroupACL }}`
- Other: `{{ .OtherACL }}`
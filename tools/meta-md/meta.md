# {{ .Filename }}

## Attributes

- Size: `{{ .SizeHuman }}` (`{{ .SizeBytes }}` bytes)
- Path:
  ```
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
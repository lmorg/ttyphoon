Runs a `jq` script for easy manipulation of JSON data.
The input format MUST conform to the following XML schema:
```
<xml>
	<query>.example.jq.query</query>
	<json>{ "example": "this is example json" }</json>
</xml>
```
This will execute 'jq $1' where
- `$1` is the XML field 'query'
- and the STDIN of `jq` is the XML field 'json'
- STDOUT and STDERR are passed back to you
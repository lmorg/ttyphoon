Runs a `jq` script for easy manipulation of JSON data.
The input MUST be a JSON object with a jq query and a JSON value:
```
{
  "query": ".example.jq.query",
  "json": { "example": "this is example json" }
}
```
Pass the object directly as tool arguments; do not wrap it in an `input` field.
`query` is passed to `jq` as its filter, and `json` is serialized to JSON and
passed to `jq` on STDIN. `json` can be any valid JSON value, including an object,
array, string, number, boolean, or `null`.
- STDOUT and STDERR are passed back to you
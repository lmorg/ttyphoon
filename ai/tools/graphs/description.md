Create a chart from tabular data and return it as an image in the conversation.
Charts are saved under the application's `.images` directory and included as a
Markdown image so the user can see the result inline.

Pass a JSON object directly as the tool arguments:

```json
{
  "headings": ["Month", "Sales", "Costs"],
  "rows": [["Jan", "12", "5"], ["Feb", "18", "7"]],
  "type": "bar",
  "categoryColumn": 0,
  "valueColumns": [1, 2],
  "seriesNames": ["Sales", "Costs"],
  "title": "Monthly totals"
}
```

Supported types are `line`, `area`, `bar`, `horizontal-bar`, `scatter`, `pie`,
and `doughnut`. `categoryColumn` and each `valueColumns` entry are zero-based
indices into `headings`. Select columns explicitly; do not guess. Pie and
doughnut require exactly one value column. Scatter uses the category column for
sample positions. Numeric values are strings, matching CSV data; blank values
are treated as missing. Non-numeric values are rejected with their row and
column identified.

Charts are always saved as PNG at 800x600. Optional fields are `title`,
`seriesNames`, `categoryAxisTitle`, and `valueAxisTitle`.
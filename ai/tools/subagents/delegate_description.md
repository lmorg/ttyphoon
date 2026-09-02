Delegate a task to one or more fresh, isolated sub-agents. Each sub-agent runs
in its own tool-enabled loop with its own context window and returns a written
summary; you only see that summary, not its intermediate tool calls or reasoning.

Use this when a task will require many tool calls, will produce large
intermediate results, or is otherwise likely to consume too much of your own
context window (e.g. auditing many files, reconciling large datasets,
open-ended research). Do NOT use it for tasks that need fewer than ~3 tool
calls — handle those yourself directly.

Entries in the input array run in parallel, so independent tasks should be
split into separate entries rather than run one after another.

Input must be a JSON array with name and prompt strings:
```
[
	{ "name": "example", "prompt": "an example prompt" },
	{ "name": "another delegate", "prompt": "run this in parallel" }
]
```
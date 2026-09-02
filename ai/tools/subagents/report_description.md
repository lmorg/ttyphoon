Delegate a reporting task to one or more fresh, isolated sub-agents. Each sub-agent runs
in its own tool-enabled loop with its own context window and returns a written
report; you only see that report, not its intermediate tool calls or reasoning.

Use this when a task will require many tool calls, or multiple iterations of reasoning across one or more datasets such as tables, API calls, or open ended research that requires structured data to be returned

Entries in the input array run in parallel, so independent tasks should be split into separate entries rather than run one after another.

The data `report` returns will be passed to you verbatim without additional summarizing. 

Input must be a JSON array with name and prompt strings:
```
[
	{ "name": "example", "prompt": "an example prompt" },
	{ "name": "another delegate", "prompt": "run this in parallel" }
]
```
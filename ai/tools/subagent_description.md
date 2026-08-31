Runs a stateless sub-agents in parallel with tools explicitly allowed for parallel execution.
This is useful for shortening, investigation or research tasks. And for reducing token usage in the main agent, particularly for larger or more complex queries.
 
Input must be JSON array with name and prompt strings:
```
[
	{ "name": "example", "prompt": "an example prompt" },
	{ "name": "another subagent", "prompt": "run this in parallel" }
]
```
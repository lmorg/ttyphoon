Read supporting files from skills selected for the current request.

- Use a skill's exact `name` and a path relative to that skill directory.
- Use `.` or a directory path to list its immediate contents.
- Read text files, including references and scripts. Optional line ranges can limit large reads.
- The tool only exposes skills selected for the current request and rejects paths that escape their directories.
- For an executable script, inspect its contents first, then use the Bash tool with the absolute path returned by this tool if the skill instructions call for running it.
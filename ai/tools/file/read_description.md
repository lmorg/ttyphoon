Open a local files for reading and return their contents.
Useful for debugging output that references local files.
The output of this tool will conform to the `txtar` specification.
Any files that could not be opened will be returned with the contents saying "!!! Cannot open file"
The input for this tool MUST be a JSON array of strings. Each array item will be a file you want the contents of.
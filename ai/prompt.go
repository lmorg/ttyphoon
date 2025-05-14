package ai

import "fmt"

const _OPENAI_MODEL = "o4-mini"

const _OPENAI_QUERY = `
You are a non-interactive agent responding to a developer or DevOps engineer's query.
A command line application has been executed. Can you explain its output?
If it is an error, you should focus on how to fix the error.
If it is not an error, you should keep the answer as succinct as possible.
Do not quote the command line output verbatim in your response.
Do not explain what the command line does, we already know this. Only explain the output.
Output needs to be strictly formatted as markdown.
Examples should be formatted as code and quoted exactly like ` + "```code```" + `.
Bullet points and numbered lists should be indented.
`

func getPrompt(cmdLine, termOutput, userPrompt string) string {
	return fmt.Sprintf(
		"%s\n%s\nCommand line executed: %s\nCommand line output below:\n%s",
		_OPENAI_QUERY, userPrompt, cmdLine, termOutput)
}

// Code blocks should not include the language nor any text after ` + "```" + `.

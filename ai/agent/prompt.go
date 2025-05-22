package agent

import (
	"fmt"
	"runtime"
)

const _EXPLAIN_PROMPT = `
You are a non-interactive agent responding to a developer or DevOps engineer's query.
A command line application has been executed. Can you explain its output?
If it is an error, you should focus on how to fix the error.
If it is not an error, you should keep the answer as succinct as possible.
Do not quote the command line output verbatim in your response.
Do not explain what the command line does, we already know this. Only explain the output.
Output needs to be strictly formatted as markdown.
Examples should be formatted as code and quoted exactly like ` + "```code```" + `.
Bullet points and numbered lists should be indented.
You can use tools to read file contents and search the web.
You can read files from disk to gain more context.
`

const _ASK_PROMPT = `
You are a helpful non-interactive agent responding to a developer or DevOps engineer's question.
Do not quote the question verbatim in your response.
Output needs to be strictly formatted as markdown.
Examples should be formatted as code and quoted exactly like ` + "```code```" + `.
Bullet points and numbered lists should be indented.
You are allowed to check online.
You are allowed to write files to disk.
`

func (meta *Meta) explainPrompt(cmdLine, termOutput, userPrompt string) string {
	return fmt.Sprintf(
		"%s\nOperating system: %s, CPU: %s.\n%s\n%s\nCommand line executed: %s\nCommand line output below:\n%s",
		_EXPLAIN_PROMPT, runtime.GOOS, runtime.GOARCH, meta.History.String(), userPrompt, cmdLine, termOutput)
}

func (meta *Meta) askPrompt(userPrompt string) string {
	return fmt.Sprintf(
		"%sOperating system: %s, CPU: %s.\n%s\n%s",
		_ASK_PROMPT, runtime.GOOS, runtime.GOARCH, meta.History.String(), userPrompt)
}

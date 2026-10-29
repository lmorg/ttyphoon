Bash command and script tool. Runs the provided Bash script in a new cloned terminal pane that inherits the active pane's working directory and environment. Use it for commands requiring a TTY and composed shell work, including variables, loops, conditionals, and command substitution.

## Keep progress visible

- This is a real, visible TTY. Run progress-producing and long-running commands in the foreground so the user can watch them advance in the cloned pane.
- Do not hide, buffer, or truncate live output with pipelines or redirections. In particular, do not append `| head -n40`, `| tail`, `| grep`, or similar filters to a running command just to limit what the model receives. Programs often change their buffering when stdout is a pipe, which can make a command appear stalled; `head` can also close the pipe early.
- Do not combine streams with `2>&1`; stdout and stderr are both visible in the terminal.
- Large tool results may be summarized automatically before they are returned to the model (the default summarization threshold is 8,192 characters). Do not crop useful output just to save context. The user can inspect the live pane; the tool returns a final terminal-screen snapshot, not a complete stdout/stderr log or a live AI-panel stream.
- If output must also be saved, preserve live visibility and use `tee` only when it does not cause the command to buffer or interfere with its interactive behavior. Avoid redirecting the only visible output to a file.

## Input and execution

- Pass a JSON object with the script in `script` and a concise one-line purpose in `summary`, for example `{"summary":"Print a greeting to verify Bash execution","script":"set -e\nprintf '%s\\n' hello"}`. Do not wrap the object in an `input` property.
- The summary is displayed to the user when approval is required. Describe what the command accomplishes, including important side effects. It is explanatory only and is never executed; the full script remains visible for review.
- The original pane is not modified. Shell syntax is Bash regardless of the original pane's shell.
- Execution/setup failures are returned in `error`; a non-zero `exitCode` means the script failed. Correct the cause rather than assuming success.
- If interactive input is required, the user can respond directly in the cloned terminal pane. Do not request secrets through chat.
- Cancelling the tool stops waiting; it does not necessarily terminate the cloned command. The pane can be inspected or killed through its normal controls.
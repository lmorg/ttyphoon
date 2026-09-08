Command-line tool. Sends the provided text directly to the active terminal as if typed by the user.
The output block is then passed to you after the command has been executed.
ANSI escape sequences are not included in the returned output.
The returned output will be the final visual state of the terminal command output. For example where ANSI escape sequences are used to move the curser, and text then overwrites previous text, such as with package installers or interactive menus, the preview characters wouldn't be in the final visual state and thus not in the returned output.
Commands will run on the active shell. You should take care to write command lines that conform to the programming language syntax of that shell. The shell name is included below.
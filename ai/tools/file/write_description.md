Creates new files.
Useful for writing new code and configuration.
Use the `patchFile` or `insertLines` tool to edit a file that already exists: this tool overwrites the whole file, which wastes tokens and risks discarding content you did not intend to change.
File contents should contain the entire file.
The input of this tool MUST conform to the `txtar` specification.

## `txtar` Specification

A txtar archive is zero or more comment lines and then a sequence of file entries. Each file entry begins with a file marker line of the form "-- FILENAME --" and is followed by zero or more file content lines making up the file data. The comment or file content ends at the next file marker line. The file marker line must begin with the three-byte sequence "-- " and end with the three-byte sequence " --", but the enclosed file name can be surrounding by additional white space, all of which is stripped.

If the txtar file is missing a trailing newline on the final line, parsers should consider a final newline to be present anyway.

There are no possible syntax errors in a txtar archive.

### `txtar` Example

```
-- hello world.txt --
this is the contents of the "hello world.txt" text file
-- example.sh --
# this is the contents of the "example.sh" shell script
```
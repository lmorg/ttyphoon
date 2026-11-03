Rename or move a regular file or directory inside the active project.

- Input fields: `source` and `destination`.
- Missing destination parent directories are created after approval.
- Existing destinations are never overwritten.
- Symlinks and the project root cannot be renamed.
- Directories can be moved only outside themselves.
- This tool is enabled by default and requires user approval.
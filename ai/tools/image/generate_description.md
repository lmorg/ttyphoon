Generates an image from a text description and saves it into the workspace.
Can also edit/modify an existing image when `inputImage` is given.

Use this tool when the user asks for a picture, diagram, illustration, icon,
logo, mockup or any other generated image — or asks you to change, edit,
recolour, extend or otherwise modify an image that already exists.

The input for this tool MUST be a JSON object passed directly as tool
arguments. Do not wrap it in an `input` property:

```json
{
  "prompt": "a watercolour painting of a lighthouse at dusk",
  "size": "1024x1024",
  "quality": "high"
}
```

Rules:

- `prompt` is required. Describe the image in detail: subject, style, colours,
  composition and mood all improve the result. When `inputImage` is set,
  describe the change you want made, not the whole scene from scratch.
- `inputImage` is optional. Set it to the path of an existing image (relative
  to the current project root, or an absolute path) to edit that image instead
  of generating a new one. The path must be inside the workspace or the
  application's `.images` directory, which contains generated images and
  pasted/uploaded attachments; anything else is rejected.
- For an inline user attachment, use the exact `inputImage` path supplied beside
  that image in the request. It is already saved and accessible to this tool.
  Do not run Bash, search directories, or copy it into the workspace to locate it.
- `file` is optional and should be **omitted** in almost every case. Only set it
  if the user has explicitly asked for the image to be saved at a particular
  name or path. Do not invent a filename. When omitted, the image is saved to a
  timestamped `.png` under `~/Documents/ttyphoon/.images/` automatically.
- When the user does specify `file`, it is relative to the working directory and
  must stay within it. Any parent directories are created for you. An existing
  file will NOT be overwritten; the call fails instead.
- `size` is optional. Accepted values are `1024x1024` (square, the default),
  `1536x1024` (landscape), `1024x1536` (portrait) and `auto`.
- `quality` is optional and may be `low`, `medium`, `high` or `auto`.

Image generation is slow and costs significantly more than a text response, so
generate one image per request unless the user explicitly asks for variations.

On success the tool returns the path the image was written to. Reference that
exact path in your reply as markdown, for example
`![lighthouse](/home/user/Documents/ttyphoon/.images/generated-image-20260826-140301.png)`,
so the user can see it inline.

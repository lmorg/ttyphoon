# 0029 - AI image attachments render in the stream

## Status

Accepted.

## Context

The Notes image context menu can ask the agent about an image. The model needs
that image as multimodal input, but the AI panel also needs to show the image
that was attached. Passing a `data:image/...;base64,...` URL through the visible
prompt or session metadata caused the base64 payload to be rendered into the
AI stream and could overwhelm the frontend.

The existing AI prompt path already represents images as Eino multimodal
`ImageURL` message parts using `ImageAttachment{MIMEType, Base64}`. That payload
must remain private to the model request. Separately, the panel needs a normal
file-backed Markdown reference that can be loaded by the Notes image renderer.

## Decision

`AskAIImage` is a dedicated Wails entry point. It:

1. validates the incoming data URL;
2. decodes and saves the image under `~/Documents/ttyphoon/.images/` with a unique
   `uploaded-image-<timestamp>.<extension>` name;
3. passes the original decoded image data to `ai.ExplainDoc` as an
   `ImageAttachment` for the model;
4. stores only this display-safe metadata in the session output:

   ```text
   Image attachment: <original filename>

   ![uploaded <saved path>](<saved path>)
   ```

The session-log prefix formatter recognizes Markdown image lines in an output
block and emits them outside the text code fence. Consequently, the attachment
text remains readable and the following image is rendered by the AI panel.

Base64 is never written to `Meta.OutputBlock`, session history, session-log
Markdown, or the `aiResponseStream` event.

The Notes frontend already resolves file-backed image paths through `GetImage`,
so no data URL is needed in the stream.

### Amendment: the generic document path had the same gap

`askAIAboutCurrentDocument()` (the "Ask AI..." item present on most content
context menus) builds its context from `getCurrentDocumentContentsForAI()`,
which read `elements.imageViewImg.src` — a `data:image/...;base64,...` URL —
directly as document text when the open document was an image. That text
became `Meta.OutputBlock` and was persisted the same way the original bug
described above was, just via a different call site. `askAIAboutCurrentDocument`
now special-cases `fileType === 'image'` and routes through the same
`askAIAboutImage()` bridge used by the image context menu, so there is a single
path for turning an on-screen image into an AI request. See ADR 0031 for the
defensive render-side truncation that also protects against any base64 that
was already persisted before this fix.

### Amendment: opt-in inputbox image paste

Updated 2026-10-21: `InputBoxWTOptions.AllowImagePaste` enables clipboard image
attachments and defaults to false. AI Ask, skill, URI, command explanation and
document explanation dialogs enable it. Confirmation dialogs always disable it.

The inputbox accepts browser clipboard files with a native `GetClipboardData`
fallback, shows removable previews, and disables submission while reads are
pending. Cancelling or reopening discards pending results and attachments. Text
paste retains its normal behavior. Images are not stored in inputbox history.

Callbacks carry structured `Images` independently from text and variables. The
backend retains the opt-in with the callback, strips images for disabled or
cancelled dialogs, and validates enabled submissions. Accept at most six PNG,
JPEG, GIF or WebP images, each at most 8 MiB and 40 megapixels. Validation checks
decoded headers and declared MIME types, not just filename extensions.

AI submission stores pasted images under the shared application `.images`
directory with unique names and sends their bytes through Eino multimodal image
parts. Only escaped file-backed Markdown references are added to visible
metadata, preserving the original context and avoiding base64 in prompt logs.
Draft saving and expanded variable history are separate features.

### Amendment: model-visible local attachment references

Updated 2026-10-22: image bytes alone do not tell the model which saved file can
be passed to an image-editing tool. Before each image part that has a saved
`Path`, emit a text part containing a JSON reference with its 1-based attachment
number and exact `inputImage` path. JSON encoding preserves spaces, quotes and
other path characters without confusing them with Markdown or URL encoding.

The reference instructs the model to use that path directly with `generateImage`
for requested edits, without Bash, filesystem searches or workspace copies.
The tool description explicitly accepts pasted/uploaded files in the existing
application `.images` directory. No filesystem access rules are broadened.
Pathless attachments and text-only messages retain their previous behavior.
Model path references contain no inline image bytes; display-safe file-backed
Markdown metadata and the original multimodal image bytes remain unchanged.

## Consequences

- The model receives a real multimodal image upload.
- The user sees the uploaded image immediately after the attachment metadata.
- Session logs remain compact and renderable instead of containing base64 blobs.
- Uploaded images are retained in `~/Documents/ttyphoon/.images/`; automatic cleanup is a
   separate lifecycle concern and is not part of this decision.
- The image display path is absolute and file-backed, so historical prompt logs
  can render it after the current request completes.

## References

- `frontend.go` - `AskAIImage`, `saveAIImageUpload`
- `ai/prompts/prompts.go` - `buildUserMessageWithImages`
- `ai/agent/sessiondb/session_log.go` - `writeSessionOutput`
- `frontend/src/notes.js` - image context menu and `AskAIImage` call

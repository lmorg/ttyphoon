package menuhyperlink

import (
	"io"
	"os"

	"github.com/lmorg/ttyphoon/types"
	"golang.design/x/clipboard"
)

func copyLinkToClipboard(renderer types.Renderer, url string) {
	renderer.DisplayNotification(types.NOTIFY_INFO, "Link copied to clipboard")
	clipboard.Write(clipboard.FmtText, []byte(url))
}

const _CONTENTS_CLIP_MAX = 10 * 1024 * 1024 // 10 MB

func copyContentsToClipboard(renderer types.Renderer, path string) {
	f, err := os.Open(path)
	if err != nil {
		renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	if info.Size() > _CONTENTS_CLIP_MAX {
		renderer.DisplayNotification(types.NOTIFY_WARN, "file too large")
		return
	}

	b, err := io.ReadAll(f)
	if err != nil {
		renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	renderer.DisplayNotification(types.NOTIFY_INFO, "File contents copied to clipboard")
	clipboard.Write(clipboard.FmtText, b)
}

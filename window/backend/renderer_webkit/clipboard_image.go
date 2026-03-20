package rendererwebkit

import (
	"errors"

	"github.com/lmorg/ttyphoon/types"
	"golang.design/x/clipboard"
)

func (wr *webkitRender) CopyImageToClipboard(png []byte) error {
	if len(png) == 0 {
		return errors.New("empty image data")
	}

	webkitClipboardInit.Do(func() { _ = clipboard.Init() })
	clipboard.Write(clipboard.FmtImage, png)

	if wr != nil {
		wr.DisplayNotification(types.NOTIFY_INFO, "Copied image to clipboard")
	}

	return nil
}

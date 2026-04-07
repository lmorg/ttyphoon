package menuhyperlink

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/lmorg/ttyphoon/types"
)

func getVar(link *link) func(string) string {
	return func(s string) string {
		switch s {
		case "url":
			return link.url
		case "scheme":
			return link.scheme
		case "path":
			return link.path
		default:
			return s
		}
	}
}

func schemaOrPath(link *link) string {
	if link.scheme == "file" {
		return link.path
	} else {
		return link.url
	}
}

func OpenWith(renderer types.Renderer, url, label string, exe []string) {
	link := makeLink(renderer, url, label)
	if link.url == "" {
		return
	}

	var b []byte
	buf := bytes.NewBuffer(b)

	for param := range exe {
		exe[param] = os.Expand(exe[param], getVar(link))
	}

	cmd := exec.Command(exe[0], exe[1:]...)
	cmd.Stderr = buf

	err := cmd.Start()
	if err != nil {
		link.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			msg := buf.String()
			if msg == "" {
				msg = err.Error()
			}
			link.renderer.DisplayNotification(types.NOTIFY_ERROR, msg)
		}
	}()
}

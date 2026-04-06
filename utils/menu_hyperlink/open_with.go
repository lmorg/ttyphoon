package menuhyperlink

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/lmorg/ttyphoon/types"
)

func getVar(link linkT) func(string) string {
	return func(s string) string {
		switch s {
		case "url":
			return link.Url()
		case "scheme":
			return link.Scheme()
		case "path":
			return link.Path()
		default:
			return s
		}
	}
}

func schemaOrPath(link linkT) string {
	if link.Scheme() == "file" {
		return link.Path()
	} else {
		return link.Url()
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
		link.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			msg := buf.String()
			if msg == "" {
				msg = err.Error()
			}
			link.Renderer().DisplayNotification(types.NOTIFY_ERROR, msg)
		}
	}()
}

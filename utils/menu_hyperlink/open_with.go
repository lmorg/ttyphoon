package menuhyperlink

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/lmorg/ttyphoon/types"
)

type LinkT interface {
	Renderer() types.Renderer
	Url() string
	Scheme() string
	Path() string
	Label() string
}

func getVar(link LinkT) func(string) string {
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

func schemaOrPath(link LinkT) string {
	if link.Scheme() == "file" {
		return link.Path()
	} else {
		return link.Url()
	}
}

func OpenWith(link LinkT, exe []string) {
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

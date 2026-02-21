package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/lmorg/ttyphoon/utils/exit"
)

var (
	fAnsiImage string
	fWindow    string
	fMeta      string
)

const (
	ANSI_BEGIN  = "\x1b_begin;%s;%s\x1b\\"
	ANSI_END    = "\x1b_end;%s\x1b\\"
	ANSI_INSERT = "\x1b_insert;%s;%s\x1b\\"
)

func getFlags() {
	flag.StringVar(&fAnsiImage, "image", "", "")
	flag.StringVar(&fWindow, "window", "", "window to display")
	flag.StringVar(&fMeta, "meta", "", "payload")
	flag.Parse()

	switch {
	case fAnsiImage != "":
		ansiImage()
	}
}

func ansiImage() {
	_, exists := os.LookupEnv("SSH_TTY")
	if exists {

		f, err := os.Open(fAnsiImage)
		die(err)

		b, err := io.ReadAll(f)
		die(err)

		b64 := base64.StdEncoding.EncodeToString(b)

		die(err)
		fmt.Printf(ANSI_INSERT, "image", params(map[string]any{
			"base64": b64,
		}))
		exit.Exit(0)
	}

	fmt.Printf(ANSI_INSERT, "image", params(map[string]any{
		"filename": fAnsiImage,
	}))
	exit.Exit(0)
}

func die(err error) {
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		exit.Exit(1)
	}
}

func params(params map[string]any) string {
	b, err := json.Marshal(params)
	die(err)
	return string(b)
}

package dispatcher

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
)

type PayloadT struct {
	Window     WindowStyleT `json:"window"`
	Parameters any          `json:"parameters"`
}

type WindowStyleT struct {
	Fg          types.Colour `json:"fg"`
	Bg          types.Colour `json:"bg"`
	Selection   types.Colour `json:"Selection"`
	Pos         types.XY     `json:"pos"`
	Size        types.XY     `json:"size"`
	AlwaysOnTop bool         `json:"alwaysOnTop"`
	Frameless   bool         `json:"frameLess"`
	FontFamily  string       `json:"fontFamily"`
	FontSize    int          `json:"fontSize"`
}

func NewWindowStyle() *WindowStyleT {
	fontFamily := config.Config.TypeFace.FontName
	if fontFamily == "" {
		fontFamily = "Hasklig"
	}
	return &WindowStyleT{
		Fg:         *types.SGR_DEFAULT.Fg,
		Bg:         *types.SGR_DEFAULT.Bg,
		Selection:  *types.COLOR_SELECTION,
		Pos:        types.XY{},
		Size:       types.XY{X: 1024, Y: 768},
		FontFamily: fmt.Sprintf(`"%s", monospace`, fontFamily),
		FontSize:   config.Config.TypeFace.FontSize,
	}
}

func DisplayWindow[P PInputBoxT | PMarkdownT](windowName WindowTypeT, windowStyle *WindowStyleT, parameters P, response any, callback func(error)) func() {
	payload := &PayloadT{
		Window:     *windowStyle,
		Parameters: parameters,
	}

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		callback(err)
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		callback(err)
		return nil
	}

	cmd := exec.Command(exe)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("%s=%s", ENV_WINDOW, windowName),
		fmt.Sprintf("%s=%s", ENV_PARAMETERS, string(payloadJson)),
	)

	err = cmd.Start()
	if err != nil {
		callback(err)
		return nil
	}

	go func() {
		err = cmd.Wait()
		if err != nil {
			// don't report error because we might have terminated the process
			return
		}

		if stderr.Len() > 0 {
			callback(errors.New(stderr.String()))
			return
		}
		err = json.Unmarshal(stdout.Bytes(), response)
		callback(err)
	}()

	return func() {
		_ = cmd.Process.Kill()
	}
}

func GetPayload(payload *PayloadT) error {
	params := os.Getenv(ENV_PARAMETERS)
	if params == "" {
		payload.Window = *NewWindowStyle()
		return nil
	}
	return json.Unmarshal([]byte(params), payload)
}

func Response(response any) error {
	// we don't care about errors here
	b, err := json.Marshal(response)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(b)
	if err != nil {
		return err
	}

	return nil
}

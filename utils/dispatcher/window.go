package dispatcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	Pos         types.XY  `json:"pos"`
	Size        types.XY  `json:"size"`
	AlwaysOnTop bool      `json:"alwaysOnTop"`
	Frameless   bool      `json:"frameLess"`
	FontFamily  string    `json:"fontFamily"`
	FontSize    int       `json:"fontSize"`
	Title       string    `json:"title"`
	Colours     *ColoursT `json:"colors"`
}

type ColoursT struct {
	Fg            types.Colour `json:"fg"`
	Bg            types.Colour `json:"bg"`
	Black         types.Colour `json:"black"`
	Red           types.Colour `json:"red"`
	Green         types.Colour `json:"green"`
	Yellow        types.Colour `json:"yellow"`
	Blue          types.Colour `json:"blue"`
	Magenta       types.Colour `json:"magenta"`
	Cyan          types.Colour `json:"cyan"`
	White         types.Colour `json:"white"`
	BlackBright   types.Colour `json:"blackBright"`
	RedBright     types.Colour `json:"redBright"`
	GreenBright   types.Colour `json:"greenBright"`
	YellowBright  types.Colour `json:"yellowBright"`
	BlueBright    types.Colour `json:"blueBright"`
	MagentaBright types.Colour `json:"magentaBright"`
	CyanBright    types.Colour `json:"cyanBright"`
	WhiteBright   types.Colour `json:"whiteBright"`
	Selection     types.Colour `json:"selection"`
	Link          types.Colour `json:"link"`
	Error         types.Colour `json:"error"`
}

func NewWindowStyle() *WindowStyleT {
	fontFamily := config.Config.TypeFace.FontName
	if fontFamily == "" {
		fontFamily = "Hasklig"
	}
	return &WindowStyleT{
		Colours: &ColoursT{
			Fg:            *types.SGR_DEFAULT.Fg,
			Bg:            *types.SGR_DEFAULT.Bg,
			Black:         *types.SGR_COLOR_BLACK,
			Red:           *types.SGR_COLOR_RED,
			Green:         *types.SGR_COLOR_GREEN,
			Yellow:        *types.SGR_COLOR_YELLOW,
			Blue:          *types.SGR_COLOR_BLUE,
			Magenta:       *types.SGR_COLOR_MAGENTA,
			Cyan:          *types.SGR_COLOR_CYAN,
			White:         *types.SGR_COLOR_WHITE,
			BlackBright:   *types.SGR_COLOR_BLACK_BRIGHT,
			RedBright:     *types.SGR_COLOR_RED_BRIGHT,
			GreenBright:   *types.SGR_COLOR_GREEN_BRIGHT,
			YellowBright:  *types.SGR_COLOR_YELLOW_BRIGHT,
			BlueBright:    *types.SGR_COLOR_BLUE_BRIGHT,
			MagentaBright: *types.SGR_COLOR_MAGENTA_BRIGHT,
			CyanBright:    *types.SGR_COLOR_CYAN_BRIGHT,
			WhiteBright:   *types.SGR_COLOR_WHITE_BRIGHT,
			Selection:     *types.COLOR_SELECTION,
			Link:          *types.SGR_COLOR_BLUE,
			Error:         *types.COLOR_ERROR,
		},
		Pos:        types.XY{},
		Size:       types.XY{X: 1024, Y: 768},
		FontFamily: fmt.Sprintf(`"%s", monospace`, fontFamily),
		FontSize:   config.Config.TypeFace.FontSize,
	}
}

func DisplayWindow[P PInputBoxT | PMarkdownT](windowName WindowTypeT, windowStyle *WindowStyleT, parameters *P, callback RespFunc) (*IpcT, func()) {
	payload := &PayloadT{
		Window:     *windowStyle,
		Parameters: parameters,
	}

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		callback(&IpcMessageT{Error: err})
		return nil, nil
	}

	exe, err := os.Executable()
	if err != nil {
		callback(&IpcMessageT{Error: err})
		return nil, nil
	}

	cmd := exec.Command(exe)
	cmd.Stdin = bytes.NewBuffer(payloadJson)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("%s=%s", ENV_WINDOW, windowName),
	)

	err = cmd.Start()
	if err != nil {
		callback(&IpcMessageT{Error: err})
		return nil, nil
	}

	cleanUp := func() { _ = cmd.Process.Kill() }
	cleanUpFuncs = append(cleanUpFuncs, cleanUp)
	return hostListen(callback), cleanUp
}

func GetPayload(payload *PayloadT) error {
	var err error

	b := []byte(os.Getenv(ENV_PARAMETERS))

	if len(b) == 0 {
		b, err = io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
	}

	if len(b) == 0 {
		payload.Window = *NewWindowStyle()
		return nil
	}

	err = json.Unmarshal(b, payload)
	if err != nil {
		return err
	}

	if payload.Window.Colours == nil {
		payload.Window.Colours = NewWindowStyle().Colours
	}

	return nil
}

package iterm2

import (
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
)

// GetTheme loads an iTerm2 .plist theme and returns a map of colors
func GetTheme(filename string) error {
	// Open the plist file
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	theme, err := unmarshalTheme(file)
	if err != nil {
		return err
	}

	return convertToMxttyTheme(theme)
}

func convertToMxttyTheme(theme map[string]Color) error {
	for name, color := range theme {
		var err error
		switch name {
		case "Ansi 0 Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_BLACK, 255)
		case "Ansi 1 Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_RED, 255)
		case "Ansi 2 Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_GREEN, 255)
		case "Ansi 3 Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_YELLOW, 255)
		case "Ansi 4 Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_BLUE, 255)
		case "Ansi 5 Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_MAGENTA, 255)
		case "Ansi 6 Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_CYAN, 255)
		case "Ansi 7 Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_WHITE, 255)
		case "Ansi 8 Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_BLACK_BRIGHT, 255)
		case "Ansi 9 Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_RED_BRIGHT, 255)
		case "Ansi 10 Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_GREEN_BRIGHT, 255)
		case "Ansi 11 Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_YELLOW_BRIGHT, 255)
		case "Ansi 12 Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_BLUE_BRIGHT, 255)
		case "Ansi 13 Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_MAGENTA_BRIGHT, 255)
		case "Ansi 14 Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_CYAN_BRIGHT, 255)
		case "Ansi 15 Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_WHITE_BRIGHT, 255)
		case "Background Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_BACKGROUND, 255)
		case "Foreground Color":
			err = rgbRealToByteColor(color, types.SGR_COLOR_FOREGROUND, 255)
		case "Selection Color":
			err = rgbRealToByteColor(color, types.COLOR_SELECTION, 128)

		default:
			debug.Log("skipping component: " + name)
		}

		if err != nil {
			return fmt.Errorf("invalid component '%s': %v", name, err)
		}
	}

	types.THEME_LIGHT = (float64(types.SGR_COLOR_BACKGROUND.Red)+
		float64(types.SGR_COLOR_BACKGROUND.Green)+
		float64(types.SGR_COLOR_BACKGROUND.Blue))/3 > 128

	var shadowMod float64 = 3
	if types.THEME_LIGHT {
		shadowMod = 2
		types.COLOR_TEXT_SHADOW.Alpha = 192
	} else {
		types.COLOR_TEXT_SHADOW.Alpha = 255
	}
	types.COLOR_TEXT_SHADOW.Red = byte(float64(types.SGR_COLOR_BACKGROUND.Red) / shadowMod)
	types.COLOR_TEXT_SHADOW.Green = byte(float64(types.SGR_COLOR_BACKGROUND.Green) / shadowMod)
	types.COLOR_TEXT_SHADOW.Blue = byte(float64(types.SGR_COLOR_BACKGROUND.Blue) / shadowMod)

	//types.COLOR_SEARCH_RESULT.Alpha = 128

	return nil
}

func rgbRealToByteColor(rCol Color, bCol *types.Colour, alpha byte) error {
	if rCol.Red > 1 || rCol.Green > 1 || rCol.Blue > 1 {
		return errors.New("rgb value > 1")
	}

	if rCol.Red < 0 || rCol.Green < 0 || rCol.Blue < 0 {
		return errors.New("rgb value < 0")
	}

	bCol.Red = byte(math.Round(rCol.Red * 255))
	bCol.Green = byte(math.Round(rCol.Green * 255))
	bCol.Blue = byte(math.Round(rCol.Blue * 255))
	bCol.Alpha = byte(math.Round(rCol.Alpha * 255))
	if bCol.Alpha == 0 {
		bCol.Alpha = alpha
	}

	return nil
}

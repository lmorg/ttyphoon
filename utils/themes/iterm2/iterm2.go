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
	for name, colour := range theme {
		var err error
		switch name {
		case "Ansi 0 Color":
			err = rgbRealToByteColour(colour, types.SGR_COLOUR_BLACK)
		case "Ansi 1 Color":
			err = rgbRealToByteColour(colour, types.SGR_COLOUR_RED)
		case "Ansi 2 Color":
			err = rgbRealToByteColour(colour, types.SGR_COLOUR_GREEN)
		case "Ansi 3 Color":
			err = rgbRealToByteColour(colour, types.SGR_COLOUR_YELLOW)
		case "Ansi 4 Color":
			err = rgbRealToByteColour(colour, types.SGR_COLOUR_BLUE)
		case "Ansi 5 Color":
			err = rgbRealToByteColour(colour, types.SGR_COLOUR_MAGENTA)
		case "Ansi 6 Color":
			err = rgbRealToByteColour(colour, types.SGR_COLOUR_CYAN)
		case "Ansi 7 Color":
			err = rgbRealToByteColour(colour, types.SGR_COLOUR_WHITE)
		case "Ansi 8 Color":
			err = rgbRealToByteColour(colour, types.SGR_COLOUR_BLACK_BRIGHT)
		case "Ansi 9 Color":
			err = rgbRealToByteColour(colour, types.SGR_COLOUR_RED_BRIGHT)
		case "Ansi 10 Color":
			err = rgbRealToByteColour(colour, types.SGR_COLOUR_GREEN_BRIGHT)
		case "Ansi 11 Color":
			err = rgbRealToByteColour(colour, types.SGR_COLOUR_YELLOW_BRIGHT)
		case "Ansi 12 Color":
			err = rgbRealToByteColour(colour, types.SGR_COLOUR_BLUE_BRIGHT)
		case "Ansi 13 Color":
			err = rgbRealToByteColour(colour, types.SGR_COLOUR_MAGENTA_BRIGHT)
		case "Ansi 14 Color":
			err = rgbRealToByteColour(colour, types.SGR_COLOUR_CYAN_BRIGHT)
		case "Ansi 15 Color":
			err = rgbRealToByteColour(colour, types.SGR_COLOUR_WHITE_BRIGHT)
		default:
			debug.Log("skipping component: " + name)
		}

		if err != nil {
			return fmt.Errorf("invalid component '%s': %v", name, err)
		}
	}

	return nil
}

func rgbRealToByteColour(rCol Color, bCol *types.Colour) error {
	if rCol.Red > 1 || rCol.Green > 1 || rCol.Blue > 1 {
		return errors.New("rgb value > 1")
	}

	if rCol.Red < 0 || rCol.Green < 0 || rCol.Blue < 0 {
		return errors.New("rgb value < 0")
	}

	bCol.Red = byte(math.Round(rCol.Red * 256))
	bCol.Green = byte(math.Round(rCol.Green * 256))
	bCol.Blue = byte(math.Round(rCol.Blue * 256))
	return nil
}

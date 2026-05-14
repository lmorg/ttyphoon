package rendererwebkit

import (
	"math"

	"github.com/lmorg/ttyphoon/types"
)

const (
	compactOpCell int32 = iota + 1
	compactOpFrame
	compactOpHighlight
	compactOpRectColour
	compactOpBlockChrome
	compactOpGaugeH
	compactOpGaugeV
	compactOpTileOverlay
	compactOpImage
	compactOpTable
)

type DrawOpTuple []any

func encodeDrawCommands(commands []DrawCommand) []DrawOpTuple {
	ops := make([]DrawOpTuple, 0, len(commands))
	for i := range commands {
		cmd := commands[i]
		switch cmd.Op {
		case DrawOpCell:
			ops = append(ops, DrawOpTuple{
				compactOpCell,
				cmd.X,
				cmd.Y,
				cmd.Width,
				cmd.Char,
				cmd.Flags,
				packColour24(cmd.Fg),
				packColour24(cmd.Bg),
				packColour24(cmd.UlC),
			})

		case DrawOpFrame:
			ops = append(ops, DrawOpTuple{compactOpFrame, cmd.X, cmd.Y, cmd.Width, cmd.Height})

		case DrawOpHighlight:
			ops = append(ops, DrawOpTuple{compactOpHighlight, cmd.X, cmd.Y, cmd.Width, cmd.Height, packColour24(cmd.Fg), packColour24(cmd.Bg)})

		case DrawOpRectColour:
			ops = append(ops, DrawOpTuple{compactOpRectColour, cmd.X, cmd.Y, cmd.Width, cmd.Height, packColour24(cmd.Bg)})

		case DrawOpBlockChrome:
			ops = append(ops, DrawOpTuple{compactOpBlockChrome, cmd.X, cmd.Y, cmd.Height, cmd.EndX, packColour24(cmd.Fg)})

		case DrawOpGaugeH:
			ops = append(ops, DrawOpTuple{compactOpGaugeH, cmd.X, cmd.Y, cmd.Width, cmd.Value, cmd.Max, packColour24(cmd.Fg)})

		case DrawOpGaugeV:
			ops = append(ops, DrawOpTuple{compactOpGaugeV, cmd.X, cmd.Y, cmd.Height, cmd.Value, cmd.Max, packColour24(cmd.Fg)})

		case DrawOpTileOverlay:
			ops = append(ops, DrawOpTuple{compactOpTileOverlay, cmd.X, cmd.Y, cmd.Width, cmd.Height, cmd.Alpha})

		case DrawOpImage:
			ops = append(ops, DrawOpTuple{
				compactOpImage,
				cmd.X,
				cmd.Y,
				cmd.Width,
				cmd.Height,
				cmd.ImageID,
				cmd.SrcWidth,
				cmd.SrcHeight,
				packScale1000(cmd.SrcScaleX),
				packScale1000(cmd.SrcScaleY),
			})

		case DrawOpTable:
			ops = append(ops, DrawOpTuple{compactOpTable, cmd.X, cmd.Y, cmd.Height, cmd.Width, packColour24(cmd.Fg), cmd.Boundaries})
		}
	}

	return ops
}

func packColour24(c *types.Colour) int32 {
	if c == nil {
		return 0
	}
	return int32(c.RGB24())
}

func packScale1000(v float64) int32 {
	if v <= 0 {
		return 0
	}
	return int32(math.Round(v * 1000.0))
}

package virtualterm

import (
	"fmt"
	"os"

	"github.com/lmorg/mxtty/ai"
	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/ai/mcp_config"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
)

func (term *Term) mxapcBegin(element types.ElementID, parameters *types.ApcSlice) {
	term._activeElement = term.renderer.NewElement(term.tile, element)
}

func (term *Term) mxapcEnd(parameters *types.ApcSlice, crlf bool) {
	if term._activeElement == nil {
		return
	}
	el := term._activeElement                 // this needs to be in this order because a
	term._activeElement = nil                 // function inside _mxapcGenerate returns
	term._mxapcGenerate(el, parameters, crlf) // without processing if _activeElement set
}

func (term *Term) mxapcInsert(element types.ElementID, parameters *types.ApcSlice) {
	term._mxapcGenerate(term.renderer.NewElement(term.tile, element), parameters, true)
}

func (term *Term) _mxapcGenerate(el types.Element, parameters *types.ApcSlice, crlf bool) {
	err := el.Generate(parameters, term.sgr)
	if err != nil {
		term.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	size := el.Size()
	lineWrap := term._noAutoLineWrap
	term._noAutoLineWrap = true

	/*if !crlf {
		term._curPos.X++ // I don't know why I need this and I feel dirty adding it!
	}*/

	elPos := new(types.XY)
	for ; elPos.Y < size.Y; elPos.Y++ {
		if term.curPos().X != 0 && crlf {
			term.carriageReturn()
			term.lineFeed()
		}
		for elPos.X = 0; elPos.X < size.X && term._curPos.X < term.size.X; elPos.X++ {
			term.writeCell(types.SetElementXY(elPos), el)
		}
	}

	/*if !crlf {
		term._curPos.X -= 2 // I don't know why I need this and I feel dirty adding it!
	}*/

	term._noAutoLineWrap = lineWrap
}

func (term *Term) mxapcBeginOutputBlock(apc *types.ApcSlice) {
	debug.Log(apc)

	if term.IsAltBuf() {
		return
	}

	//if term._blockMeta == nil {
	term._blockMeta = new(types.BlockMeta)
	//}
	//if (*term.screen)[term.curPos().Y].Block == nil {
	(*term.screen)[term.curPos().Y].Block = term._blockMeta
	//}

	var params struct {
		CmdLine string
	}

	apc.Parameters(&params)

	(*term.screen)[term.curPos().Y].RowMeta.Set(types.META_ROW_BEGIN)
	(*term.screen)[term.curPos().Y].Block.Query = []rune(params.CmdLine)
}

func (term *Term) mxapcEndOutputBlock(apc *types.ApcSlice) {
	debug.Log(apc)

	if term.IsAltBuf() {
		return
	}

	pos := term.curPos()
	if pos.X == 0 {
		pos.Y--
	}
	if pos.Y < 0 {
		pos.Y = 0
	}

	var params struct {
		ExitNum  int
		MetaFlag types.BlockMetaFlag
	}

	apc.Parameters(&params)

	(*term.screen)[pos.Y].RowMeta.Set(types.META_ROW_END)
	if params.ExitNum == 0 {
		term._blockMeta.Meta.Set(types.META_BLOCK_OK | params.MetaFlag)
	} else {
		term._blockMeta.Meta.Set(types.META_BLOCK_ERROR | params.MetaFlag)
	}

	term._blockMeta.ExitNum = params.ExitNum

	// prep for new block
	term._blockMeta = new(types.BlockMeta)
}

func (term *Term) mxapcConfigExport(apc *types.ApcSlice) {
	envs := make(map[string]string)
	apc.Parameters(&envs)
	for k, v := range envs {
		err := os.Setenv(k, v)
		if err != nil {
			term.renderer.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("unable to export %s: %v", k, err))
		}
	}
}

/*func (term *Term) mxapcConfigVariables(apc *types.ApcSlice) {
	envs := make(map[string]string)
	apc.Parameters(&envs)
	for k, v := range envs {
		err := os.Setenv(k, v)
		if err != nil {
			term.renderer.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("unable to set local variable %s: %v", k, err))
		}
	}
}*/

func (term *Term) mxapcConfigUnset(apc *types.ApcSlice) {
	var envs []string
	apc.Parameters(&envs)
	for i := range envs {
		err := os.Unsetenv(envs[i])
		if err != nil {
			term.renderer.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("unable to unset %s: %v", envs[i], err))
		}
	}
}

func (term *Term) mxapcConfigMcp(apc *types.ApcSlice) {
	config := new(mcp_config.ConfigT)
	apc.Parameters(config)
	config.Source = "escape-sequence"
	go func() {
		err := ai.StartServersFromConfig(term.renderer, agent.Get(term.tile.Id()), config)
		if err != nil {
			term.renderer.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("Cannot start MCP from escape sequence: %v", err))
		}
	}()
}

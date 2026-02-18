package virtualterm

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/ai"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/file"
	historymd "github.com/lmorg/ttyphoon/utils/history_md"
)

func (term *Term) mxapcBegin(element types.ElementID, parameters *types.ApcSlice) {
	term._activeElement = term.renderer.NewElement(term.tile, element)
}

func (term *Term) mxapcEnd(parameters *types.ApcSlice) {
	if term._activeElement == nil {
		return
	}
	el := term._activeElement           // this needs to be in this order because a
	term._activeElement = nil           // function inside _mxapcGenerate returns
	term._mxapcGenerate(el, parameters) // without processing if _activeElement set
}

func (term *Term) mxapcInsert(element types.ElementID, parameters *types.ApcSlice) {
	term._mxapcGenerate(term.renderer.NewElement(term.tile, element), parameters)
}

type _apcDisplayMenuT struct {
	Title string                  `json:"title"`
	Items []*_apcDisplayMenuItemT `json:"menuItem"`
}

type _apcDisplayMenuItemT struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func (term *Term) mxapcDisplayMenu(parameters *types.ApcSlice) {
	p := new(_apcDisplayMenuT)
	err := parameters.Parameters(p)
	if err != nil {
		term.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	// TODO
}

type _mxapcDisplayInputT struct {
	Title   string   `json:"title"`
	History []string `json:"history"`
}

func (term *Term) mxapcDisplayInput(parameters *types.ApcSlice) {
	p := new(_mxapcDisplayInputT)
	err := parameters.Parameters(p)
	if err != nil {
		term.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	// TODO
}

func (term *Term) _mxapcGenerate(el types.Element, parameters *types.ApcSlice) {
	err := el.Generate(parameters)
	if err != nil {
		term.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	size := el.Size()
	lineWrap := term._noAutoLineWrap
	term._noAutoLineWrap = true

	elPos := new(types.XY)
	for ; elPos.Y < size.Y; elPos.Y++ {
		if elPos.Y > 0 {
			term.carriageReturn()
			term.lineFeed(_LINEFEED_CURSOR_MOVED)
		}
		for elPos.X = 0; elPos.X < size.X && term._curPos.X < term.size.X; elPos.X++ {
			term.writeCell(types.SetElementXY(elPos), el)
		}
	}

	term._noAutoLineWrap = lineWrap
}

func (term *Term) mxapcBeginOutputBlock(apc *types.ApcSlice) {
	debug.Log(apc)

	if term.IsAltBuf() {
		return
	}

	term._blockMeta = NewRowBlockMeta(term)
	(*term.screen)[term.curPos().Y].Block = term._blockMeta

	var params struct {
		CmdLine string
	}

	if err := apc.Parameters(&params); err != nil {
		params.CmdLine = apc.Index(2)
	}

	term._blockMeta.Query = []rune(params.CmdLine)
	(*term.screen)[term.curPos().Y].RowMeta.Set(types.META_ROW_BEGIN_BLOCK)
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

	(*term.screen)[pos.Y].RowMeta.Set(types.META_ROW_END_BLOCK)
	if params.ExitNum == 0 {
		term._blockMeta.Meta.Set(types.META_BLOCK_OK | params.MetaFlag)
	} else {
		term._blockMeta.Meta.Set(types.META_BLOCK_ERROR | params.MetaFlag)
	}

	term._blockMeta.ExitNum = params.ExitNum
	term._blockMeta.TimeEnd = time.Now()

	if config.Config.Terminal.WriteMarkdownHistory {
		var (
			screen = append(term._scrollBuf, term._normBuf...)
			begin  = int(term.curPos().Y) + len(term._scrollBuf)
			end    = begin
		)
		for ; begin >= 0; begin-- {
			if screen[begin].RowMeta.Is(types.META_ROW_BEGIN_BLOCK) {
				break
			}
		}
		go historymd.Write(term.tile, screen[max(0, begin):end])
	}

	// prep for new block
	term._blockMeta = NewRowBlockMeta(term) // TODO, wouldn't this lead to duplication of rowIDs?
}

func (term *Term) askAi(prompt string) {
	meta := agent.Get(term.tile.Id())
	insertAfterRowId := term.GetRowId(term.GetCursorPosition().Y - 1)
	ai.AskAI(meta, prompt, insertAfterRowId)
}

func (term *Term) mxapcAiAsk(parameters *types.ApcSlice) {
	prompt := parameters.Index(2)
	if prompt == "" {
		term.renderer.DisplayNotification(types.NOTIFY_DEBUG, "Missing config in `ai;agent` ANSI sequence")
		return
	}

	term.askAi(prompt)
}

type mxapcAiAgentT struct {
	mcp_config.ConfigT
	SystemPrompt string `json:"systemPrompt"`
	UserPrompt   string `json:"userPrompt"`
	//Agents       ...
}

func (term *Term) mxapcAiAgent(parameters *types.ApcSlice) {
	acpConfig := parameters.Index(2)
	if acpConfig == "" {
		term.renderer.DisplayNotification(types.NOTIFY_DEBUG, "Missing config in `ai;agent` ANSI sequence")
		return
	}

	agentConfig := new(mxapcAiAgentT)
	agentConfig.McpServers = &agentConfig.Mcp.Servers
	err := json.Unmarshal([]byte(acpConfig), agentConfig)
	if err != nil {
		term.renderer.DisplayNotification(types.NOTIFY_DEBUG, fmt.Sprintf("Cannot decode ai;agent config: %s", err))
		return
	}

	go func() {
		if len(agentConfig.Mcp.Servers) > 0 {
			err = agent.Get(term.tile.Id()).StartServersFromConfig(&agentConfig.ConfigT)
			if err != nil {
				term.renderer.DisplayNotification(types.NOTIFY_DEBUG, err.Error())
				return
			}
		}

		if agentConfig.SystemPrompt != "" {
			if strings.Contains(agentConfig.SystemPrompt, `../`) || strings.Contains(agentConfig.SystemPrompt, `..\`) {
				term.renderer.DisplayNotification(types.NOTIFY_WARN, "systemPrompt files cannot exist outside of the system-prompt directory")
			}
			f, err := file.OpenConfigFile("system-prompts", agentConfig.SystemPrompt)
			if err == nil {
				defer f.Close()
				term.renderer.DisplayNotification(types.NOTIFY_DEBUG, fmt.Sprintf("Using system prompt: %s", f.Name()))
				b, err := io.ReadAll(f)
				if err != nil {
					term.renderer.DisplayNotification(types.NOTIFY_WARN, err.Error())
					return
				}
				agentConfig.SystemPrompt = string(b)
			}
		}

		term.askAi(fmt.Sprintf("%s\n%s", agentConfig.SystemPrompt, agentConfig.UserPrompt))
	}()
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
		err := agent.Get(term.tile.Id()).StartServersFromConfig(config)
		if err != nil {
			term.renderer.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("Cannot start MCP from escape sequence: %v", err))
		}
	}()
}

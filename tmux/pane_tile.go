package tmux

import (
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/types"
)

func (p *PaneT) Name() string     { return p.title }
func (p *PaneT) SetName(s string) { p.title = s }
func (p *PaneT) Id() string       { return p.id }
func (p *PaneT) Left() int32      { return p.left }
func (p *PaneT) Top() int32       { return p.top }
func (p *PaneT) Right() int32     { return p.right }
func (p *PaneT) Bottom() int32    { return p.bottom }
func (p *PaneT) AtBottom() bool   { return p.atBottom }

func (p *PaneT) GetTerm() types.Term     { return p.term }
func (p *PaneT) SetTerm(term types.Term) { p.term = term }
func (p *PaneT) Pwd() string             { return p.curPath }

func (p *PaneT) Close() { agent.Close(p.Id()) }

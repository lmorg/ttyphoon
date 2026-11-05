package virtualterm

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/kittygraphics"
	runebuf "github.com/lmorg/ttyphoon/utils/rune_buf"
)

type kittyTestRenderer struct {
	synchronizedOutputTestRenderer
	assets []*kittyTestAsset
	ops    []string
	cells  []*types.Cell
}

type kittyTestAsset struct {
	renderer *kittyTestRenderer
	id       int
	draws    []types.GraphicsPlacement
	closes   int
}

func (r *kittyTestRenderer) LoadGraphicsImage(raw []byte) (types.GraphicsImage, error) {
	if _, err := png.Decode(bytes.NewReader(raw)); err != nil {
		return nil, err
	}
	asset := &kittyTestAsset{renderer: r, id: len(r.assets) + 1}
	r.assets = append(r.assets, asset)
	return asset, nil
}
func (r *kittyTestRenderer) GetGlyphSize() *types.XY                                          { return &types.XY{X: 10, Y: 20} }
func (r *kittyTestRenderer) DisplayNotification(types.NotificationType, string)               {}
func (r *kittyTestRenderer) GetBlinkState() bool                                              { return true }
func (r *kittyTestRenderer) DrawGaugeV(types.Tile, *types.XY, int32, int, int, *types.Colour) {}
func (r *kittyTestRenderer) DrawRectWithColour(types.Tile, *types.XY, *types.XY, *types.Colour, bool) {
	r.ops = append(r.ops, "background")
}
func (r *kittyTestRenderer) PrintCell(_ types.Tile, c *types.Cell, _ *types.XY) {
	if c.Char == ' ' && c.Sgr.Bg != nil {
		r.ops = append(r.ops, "background")
		return
	}
	r.ops = append(r.ops, "text")
	r.cells = append(r.cells, c)
}
func (r *kittyTestRenderer) PrintRow(tile types.Tile, cells []*types.Cell, pos *types.XY) {
	for _, c := range cells {
		if c != nil {
			r.PrintCell(tile, c, pos)
		}
	}
}
func (a *kittyTestAsset) DrawGraphics(_ types.Tile, g types.GraphicsPlacement) {
	a.draws = append(a.draws, g)
	a.renderer.ops = append(a.renderer.ops, fmt.Sprintf("image%d", a.id))
}
func (a *kittyTestAsset) Close() { a.closes++ }

func newKittyTestTerm(t *testing.T) (*Term, *kittyTestRenderer) {
	t.Helper()
	term := newParserTestTerm()
	term._altBuf = term.makeScreen()
	renderer := &kittyTestRenderer{}
	term.renderer = renderer
	term.tile = oscHyperlinkTestTile{term: term}
	term._hideCursor = true
	t.Cleanup(term.clearKittyGraphics)
	return term, renderer
}

func feedKitty(t *testing.T, term *Term, text string) {
	t.Helper()
	term.Pty.(*parserTestPty).FeedInput([]byte(text))
	drainMockPtyInput(t, term, len(text)*2)
}

func kittyPacket(controls string, data []byte) string {
	return "\x1b_G" + controls + ";" + base64.StdEncoding.EncodeToString(data) + "\x1b\\"
}

func TestKittyQueriesAndErrors(t *testing.T) {
	term, r := newKittyTestTerm(t)
	feedKitty(t, term, kittyPacket("a=q,i=31,s=1,v=1,f=24", []byte{0, 0, 0}))
	if got := string(term.Pty.(*parserTestPty).out); got != "\x1b_Gi=31;OK\x1b\\" {
		t.Fatalf("bad query reply %q", got)
	}

	if len(term._kitty.images) != 0 || len(r.assets) != 0 {
		t.Fatal("query stored or rendered image")
	}
	for _, tc := range []struct{ command, reply string }{
		{"a=p,i=99,p=2", "ENOENT:"},
		{"a=q,i=31,s=1,v=1,f=24,q=1;AAAA", ""},
		{"a=q,i=31,t=f;AAAA", "ENOTSUP:"},
		{"a=q,i=31,f=24,s=1,v=1;AA==", "EINVAL:"},
		{"a=q,i=31,f=24,s=1,v=1,q=2;AA==", ""},
		{"a=f,i=31", "ENOTSUP:"},
	} {
		term.Pty.(*parserTestPty).out = nil
		feedKitty(t, term, "\x1b_G"+tc.command+"\x1b\\")
		reply := string(term.Pty.(*parserTestPty).out)
		if tc.reply == "" && reply != "" || tc.reply != "" && !strings.Contains(reply, tc.reply) {
			t.Fatalf("%s: wrong reply %q", tc.command, reply)
		}
	}
}

type kittyBufferedPty struct {
	parserTestPty
	buffer *runebuf.Buf
}

func (p *kittyBufferedPty) Read() (rune, error) { return p.buffer.Read() }
func (p *kittyBufferedPty) BufSize() int        { return p.buffer.BufSize() }

func TestKittySplitTransportAndQueryOrdering(t *testing.T) {
	term, _ := newKittyTestTerm(t)
	old := config.Config.Tmux.Enabled
	config.Config.Tmux.Enabled = false
	t.Cleanup(func() { config.Config.Tmux.Enabled = old })
	pty := &kittyBufferedPty{buffer: runebuf.New()}
	t.Cleanup(pty.buffer.Close)
	term.Pty = pty
	done := make(chan error, 1)
	go func() {
		for n := 0; n < 2; n++ {
			r, err := pty.Read()
			if err != nil {
				done <- err
				return
			}
			term.readChar(r)
		}
		done <- nil
	}()
	// Split both the APC introducer and its ST terminator across writes.
	for _, part := range []string{"\x1b", "_", "G", "a=q,i=31,s=1,v=1,f=24;", "AAAA", "\x1b"} {
		pty.buffer.Write([]byte(part))
	}
	select {
	case <-done:
		t.Fatal("parser completed before split ST")
	case <-time.After(20 * time.Millisecond):
	}
	pty.buffer.Write([]byte("\\\x1b[c"))
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("split graphics transport stalled")
	}
	wantPrefix := "\x1b_Gi=31;OK\x1b\\\x1b["
	if !strings.HasPrefix(string(pty.out), wantPrefix) {
		t.Fatalf("graphics query did not precede DA response: %q", pty.out)
	}
}

func TestKittyPlacementReuseReplacementAndText(t *testing.T) {
	term, r := newKittyTestTerm(t)
	feedKitty(t, term, "ABC"+kittyPacket("a=T,i=1,p=2,s=1,v=1,f=32,c=2,r=1,C=1", []byte{255, 0, 0, 128})+"D")
	if term._curPos != (types.XY{X: 4, Y: 0}) || (*term.screen)[0].String() != "ABCD" {
		t.Fatal("graphics replaced ordinary text or moved C=1 cursor")
	}
	if len(term._kitty.placements) != 1 || len(r.assets) != 1 {
		t.Fatal("missing placement")
	}
	feedKitty(t, term, "\x1b_Ga=p,i=1,p=2,c=1,r=1,C=1\x1b\\")
	if len(term._kitty.placements) != 1 || len(r.assets) != 1 || term._kitty.placements[0].geometry.Column != 4 {
		t.Fatal("placement ID did not replace existing placement")
	}
	feedKitty(t, term, "\x1b_Ga=p,i=1,C=1\x1b\\")
	if len(term._kitty.placements) != 2 || len(r.assets) != 1 {
		t.Fatal("image not reused for anonymous placement")
	}
	feedKitty(t, term, kittyPacket("a=t,i=1,s=1,v=1,f=24", []byte{0, 0, 255}))
	if len(term._kitty.placements) != 0 || r.assets[0].closes != 1 {
		t.Fatal("retransmission did not release old image and placements")
	}
	feedKitty(t, term, "\x1b_Ga=p,i=1,C=1\x1b\\")
	if len(r.assets) != 2 {
		t.Fatal("replacement image not loaded")
	}
}

func TestKittyChunkingAndTmuxPassthrough(t *testing.T) {
	for _, depth := range []int{0, 1, 2} {
		t.Run(fmt.Sprint(depth), func(t *testing.T) {
			term, _ := newKittyTestTerm(t)
			wrap := func(text string) string {
				for n := 0; n < depth; n++ {
					text = "\x1bPtmux;" + strings.ReplaceAll(text, "\x1b", "\x1b\x1b") + "\x1b\\"
				}
				return text
			}
			feedKitty(t, term, wrap("\x1b_Ga=T,i=1,s=2,v=1,f=24,m=1,C=1;AQID\x1b\\"))
			if len(term._kitty.placements) != 0 {
				t.Fatal("partial transfer displayed")
			}
			feedKitty(t, term, wrap("\x1b_Gm=0;BAUG\x1b\\")+"X")
			if len(term._kitty.placements) != 1 || (*term.screen)[0].String() != "X" {
				t.Fatal("graphics framing leaked into text or did not complete")
			}
			if got := string(term.Pty.(*parserTestPty).out); got != "\x1b_Gi=1;OK\x1b\\" {
				t.Fatalf("reply not written to originating PTY: %q", got)
			}
		})
	}
}

func TestKittyGeometryAndLayers(t *testing.T) {
	for _, ligatures := range []bool{false, true} {
		t.Run(fmt.Sprintf("ligatures=%v", ligatures), func(t *testing.T) {
			old := config.Config.TypeFace.Ligatures
			config.Config.TypeFace.Ligatures = ligatures
			t.Cleanup(func() { config.Config.TypeFace.Ligatures = old })
			term, r := newKittyTestTerm(t)
			pixels := bytes.Repeat([]byte{255, 0, 0}, 4*2)
			for id, z := range []int{-1073741825, -1, 1} {
				feedKitty(t, term, kittyPacket(fmt.Sprintf("a=T,i=%d,s=4,v=2,f=24,x=1,y=0,w=2,h=2,c=2,r=1,X=2,Y=3,z=%d,C=1", id+1, z), pixels))
			}
			feedKitty(t, term, "\x1b[41mT")
			term.Render()
			if got := strings.Join(r.ops, ","); got != "image1,background,image2,text,image3" {
				t.Fatalf("wrong layering: %s", got)
			}
			g := r.assets[0].draws[0]
			if g.Source.Min.X != 1 || g.Source.Dx() != 2 || g.Width != 20 || g.Height != 20 || g.X != 2 || g.Y != 3 {
				t.Fatalf("wrong crop or offsets: %+v", g)
			}
			if r.cells[0].Sgr.Bg != nil || (*term.screen)[0].Cells[0].Sgr.Bg == nil {
				t.Fatal("background pass mutated terminal SGR")
			}
		})
	}
}

func TestKittyScrollResizeAlternateAndCleanup(t *testing.T) {
	term, r := newKittyTestTerm(t)
	feedKitty(t, term, kittyPacket("a=T,i=1,s=1,v=1,f=24,c=1,r=3,C=1", []byte{0, 255, 0}))
	anchor := term._kitty.placements[0].anchor
	// Move the anchor to scrollback while leaving the bottom of the placement visible.
	term._scrollBuf = append(term._scrollBuf, anchor)
	term._normBuf = append(term._normBuf[1:], term.makeRow())
	term.Render()
	if g := r.assets[0].draws[0]; g.Row != -1 {
		t.Fatalf("image did not scroll with its row: %+v", g)
	}
	term._scrollOffset = 1
	term.Render()
	if g := r.assets[0].draws[1]; g.Row != 0 {
		t.Fatal("scrollback did not restore image position")
	}
	term.Resize(&types.XY{X: term.size.X + 2, Y: term.size.Y})
	term.Render()
	if g := r.assets[0].draws[2]; g.Row != 0 || g.ClipColumns != term.size.X || g.ClipRows != term.size.Y {
		t.Fatalf("resize lost anchor or failed to update pane clipping: %+v", g)
	}
	term._scrollOffset = 0
	term.csiScreenBufferAlternative()
	term.Render()
	if len(r.assets[0].draws) != 3 {
		t.Fatal("normal image leaked into alternate screen")
	}
	feedKitty(t, term, kittyPacket("a=T,i=2,s=1,v=1,f=24,C=1", []byte{255, 0, 0}))
	term.csiScreenBufferNormal()
	if len(term._kitty.placements) != 1 {
		t.Fatal("alternate screen placement survived buffer exit")
	}
	term.eraseScrollBack()
	term.pruneKittyPlacements()
	if len(term._kitty.placements) != 0 {
		t.Fatal("evicted anchor retained placement")
	}
	term.ReleaseGraphics()
	if term._kitty != nil || r.assets[0].closes != 1 || r.assets[1].closes != 1 {
		t.Fatal("terminal cleanup leaked cached images")
	}
}

func TestKittyDeletionAndLimits(t *testing.T) {
	term, r := newKittyTestTerm(t)
	feedKitty(t, term, kittyPacket("a=T,i=1,p=2,s=1,v=1,f=24,C=1", []byte{1, 2, 3}))
	feedKitty(t, term, "\x1b_Ga=d,d=i,i=1,p=2\x1b\\")
	if len(term._kitty.placements) != 0 || len(term._kitty.images) != 1 || r.assets[0].closes != 0 {
		t.Fatal("lowercase delete freed reusable data")
	}
	feedKitty(t, term, "\x1b_Ga=p,i=1,p=3,C=1\x1b\\")
	feedKitty(t, term, "\x1b_Ga=d,d=I,i=1\x1b\\")
	if len(term._kitty.images) != 0 || r.assets[0].closes != 1 {
		t.Fatal("uppercase delete leaked image")
	}
	for id := 1; id <= kittyStoreImages+1; id++ {
		feedKitty(t, term, kittyPacket(fmt.Sprintf("a=t,i=%d,s=1,v=1,f=24,q=2", id), []byte{1, 2, 3}))
	}
	if len(term._kitty.images) != kittyStoreImages || term._kitty.images[0].id != 2 {
		t.Fatal("unused-image quota not enforced")
	}
	term._kitty.bytes = kittyStoreBytes
	for _, img := range term._kitty.images {
		term._kitty.placements = append(term._kitty.placements, &kittyPlacement{image: img, anchor: (*term.screen)[0]})
	}
	command := &kittygraphics.Command{Controls: map[string]string{"a": "t", "i": "200", "f": "24", "s": "1", "v": "1"}, Data: []byte{1, 2, 3}}
	if err := term.applyKittyCommand(command); err == nil || !strings.HasPrefix(err.Error(), "ENOSPC:") {
		t.Fatal("active images evicted instead of reporting storage quota")
	}
}

func TestKittyOversizedAndUnsupportedFraming(t *testing.T) {
	term, _ := newKittyTestTerm(t)
	feedKitty(t, term, "\x1b_G"+strings.Repeat("A", 9000)+"\x1b\\X")
	feedKitty(t, term, "\x1bPtmux;\x1b\x1b[31m\x1b\\Y")
	if got := (*term.screen)[0].String(); got != "XY" {
		t.Fatalf("untrusted escape payload leaked into text: %q", got)
	}

}

func TestKittyInvalidContinuationRepliesToOriginalImage(t *testing.T) {
	term, _ := newKittyTestTerm(t)
	feedKitty(t, term, "\x1b_Ga=T,i=42,s=2,v=1,f=24,m=1;AAAA\x1b\\")
	feedKitty(t, term, "\x1b_Gm=0,U=1;AAAA\x1b\\")
	if got := string(term.Pty.(*parserTestPty).out); !strings.Contains(got, "i=42;ENOTSUP:") {
		t.Fatalf("continuation error lost original ID: %q", got)
	}
	if term._kitty.upload != (kittygraphics.Upload{}) {
		t.Fatal("invalid continuation retained transfer")
	}
}

func TestKittyDefaultCursorAndPlacementQuota(t *testing.T) {
	term, _ := newKittyTestTerm(t)
	feedKitty(t, term, kittyPacket("a=T,i=1,s=1,v=1,f=24,c=2,r=1", []byte{1, 2, 3}))
	if term._curPos != (types.XY{X: 2, Y: 1}) {
		t.Fatalf("wrong default cursor advance: %+v", term._curPos)
	}
	for n := 1; n < kittyPlacements; n++ {
		feedKitty(t, term, "\x1b_Ga=p,i=1,C=1,q=2\x1b\\")
	}
	term.Pty.(*parserTestPty).out = nil
	feedKitty(t, term, "\x1b_Ga=p,i=1,C=1\x1b\\")
	if len(term._kitty.placements) != kittyPlacements || !strings.Contains(string(term.Pty.(*parserTestPty).out), "ENOSPC:") {
		t.Fatal("placement quota not enforced")
	}
}

func TestKittyExampleScript(t *testing.T) {
	for _, tmux := range []bool{false, true} {
		term, _ := newKittyTestTerm(t)
		command := exec.Command("bash", "../test/kitty-graphics.sh")
		if tmux {
			command.Env = []string{"PATH=/usr/bin:/bin", "TMUX=example"}
		} else {
			command.Env = []string{"PATH=/usr/bin:/bin"}
		}

		output, err := command.Output()
		if err != nil {
			t.Fatal(err)
		}
		feedKitty(t, term, string(output))
		if len(term._kitty.images) != 1 || term._kitty.images[0].bitmap.Width != 16 || term._kitty.images[0].bitmap.Height != 16 {
			t.Fatal("example did not display checkerboard")
		}
		if len(term.Pty.(*parserTestPty).out) != 0 {
			t.Fatal("quiet example injected shell input")
		}
	}
}

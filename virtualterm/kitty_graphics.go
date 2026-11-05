package virtualterm

import (
	"fmt"
	"image"
	"log"
	"math"
	"sort"
	"strings"

	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/kittygraphics"
)

const (
	kittyStoreBytes  = 128 << 20
	kittyStoreImages = 128
	kittyPlacements  = 512
)

type kittyImage struct {
	id     uint32
	bitmap *kittygraphics.Bitmap
	asset  types.GraphicsImage
}

type kittyPlacement struct {
	image         *kittyImage
	id            uint32
	anchor        *types.Row
	alt           bool
	z             int32
	geometry      types.GraphicsPlacement
	columns, rows int32
}

type kittyState struct {
	upload     kittygraphics.Upload
	images     []*kittyImage
	placements []*kittyPlacement
	bytes      int
}

func (term *Term) kittyCommand(text string) {
	if term._kitty == nil {
		term._kitty = new(kittyState)
	}
	c, err := term._kitty.upload.Push(text)
	if c == nil {
		return
	}
	if err == nil {
		term.pruneKittyPlacements()
		err = term.applyKittyCommand(c)
	}
	message := "OK"
	if err != nil {
		message = err.Error()
		log.Printf("Kitty graphics: %s", message)
	}
	if c.Number("q") == 2 || c.Number("q") == 1 && err == nil ||
		c.Number("i") == 0 && c.String("a", "t") != "q" {
		return
	}
	reply := fmt.Sprintf("\x1b_Gi=%d", c.Number("i"))
	if c.Number("p") > 0 {
		reply += fmt.Sprintf(",p=%d", c.Number("p"))
	}
	if writeErr := term.Pty.Write([]byte(reply + ";" + message + "\x1b\\")); writeErr != nil {
		term.renderer.DisplayNotification(types.NOTIFY_ERROR, "Cannot reply to Kitty graphics command: "+writeErr.Error())
	}
}

func (term *Term) applyKittyCommand(c *kittygraphics.Command) error {
	loader, ok := term.renderer.(types.GraphicsRenderer)
	if !ok {
		return fmt.Errorf("ENOTSUP:renderer does not support graphics")
	}
	action := c.String("a", "t")
	if action == "d" {
		term.deleteKittyPlacements(c)
		return nil
	}
	if action == "p" {
		if len(c.Data) != 0 {
			return fmt.Errorf("EINVAL:placement has image data")
		}
		for _, img := range term._kitty.images {
			if img.id != 0 && img.id == uint32(c.Number("i")) {
				return term.placeKittyImage(img, c, loader)
			}
		}
		return fmt.Errorf("ENOENT:image ID not found")
	}
	bitmap, err := kittygraphics.Decode(c)
	if err != nil {
		return err
	}
	if action == "q" {
		return nil
	}
	img := &kittyImage{id: uint32(c.Number("i")), bitmap: bitmap}
	// Validate the placement before replacing a previously transmitted ID.
	if action == "T" {
		if _, err = term.kittyGeometry(img, c); err != nil {
			return err
		}
	}
	if img.id != 0 {
		for _, old := range append([]*kittyImage(nil), term._kitty.images...) {
			if old.id == img.id {
				term.removeKittyImage(old)
			}
		}
	}
	for len(term._kitty.images) >= kittyStoreImages || term._kitty.bytes+bitmap.Bytes > kittyStoreBytes {
		var victim *kittyImage
		for _, candidate := range term._kitty.images {
			if !term.kittyImagePlaced(candidate) {
				victim = candidate
				break
			}
		}
		if victim == nil {
			return fmt.Errorf("ENOSPC:terminal image storage quota exceeded")
		}
		term.removeKittyImage(victim)
	}
	term._kitty.images = append(term._kitty.images, img)
	term._kitty.bytes += bitmap.Bytes
	if action == "T" {
		if err = term.placeKittyImage(img, c, loader); err != nil {
			term.removeKittyImage(img)
			return err
		}
	}
	return nil
}

func (term *Term) kittyGeometry(img *kittyImage, c *kittygraphics.Command) (*kittyPlacement, error) {
	glyph := term.renderer.GetGlyphSize()
	if glyph == nil || glyph.X <= 0 || glyph.Y <= 0 {
		return nil, fmt.Errorf("EINVAL:cell pixel dimensions unavailable")
	}
	if c.Number("X") >= int64(glyph.X) || c.Number("Y") >= int64(glyph.Y) {
		return nil, fmt.Errorf("EINVAL:pixel offset exceeds cell dimensions")
	}
	source := image.Rect(0, 0, img.bitmap.Width, img.bitmap.Height)
	x, y := int(c.Number("x")), int(c.Number("y"))
	w, h := int(c.Number("w")), int(c.Number("h"))
	if w == 0 {
		w = img.bitmap.Width - x
	}
	if h == 0 {
		h = img.bitmap.Height - y
	}
	source = source.Intersect(image.Rect(x, y, x+w, y+h))
	if source.Empty() {
		return nil, fmt.Errorf("EINVAL:source rectangle does not intersect image")
	}
	width, height := float64(source.Dx()), float64(source.Dy())
	cols, rows := c.Number("c"), c.Number("r")
	if cols > 32768 || rows > 32768 {
		return nil, fmt.Errorf("E2BIG:placement dimensions exceed limit")
	}
	if cols > 0 && rows > 0 {
		scale := math.Min(float64(cols)*float64(glyph.X)/width, float64(rows)*float64(glyph.Y)/height)
		width, height = width*scale, height*scale
	} else if cols > 0 {
		height *= float64(cols) * float64(glyph.X) / width
		width = float64(cols) * float64(glyph.X)
	} else if rows > 0 {
		width *= float64(rows) * float64(glyph.Y) / height
		height = float64(rows) * float64(glyph.Y)
	}
	if width > math.MaxInt32 || height > math.MaxInt32 {
		return nil, fmt.Errorf("E2BIG:placement pixel dimensions exceed limit")
	}
	if cols == 0 {
		cols = int64(math.Ceil((width + float64(c.Number("X"))) / float64(glyph.X)))
	}
	if rows == 0 {
		rows = int64(math.Ceil((height + float64(c.Number("Y"))) / float64(glyph.Y)))
	}
	if cols > 32768 || rows > 32768 {
		return nil, fmt.Errorf("E2BIG:placement dimensions exceed limit")
	}
	placement := &kittyPlacement{
		image: img, id: uint32(c.Number("p")), anchor: (*term.screen)[term.curPos().Y],
		alt: term.IsAltBuf(), z: int32(c.Number("z")), columns: int32(cols), rows: int32(rows),
		geometry: types.GraphicsPlacement{
			Column: term._curPos.X, X: int32(c.Number("X")), Y: int32(c.Number("Y")),
			Width: max(1, int32(math.Round(width))), Height: max(1, int32(math.Round(height))),
			Source: source,
		},
	}
	if c.Number("c") > 0 && c.Number("r") > 0 {
		placement.geometry.X += (int32(cols)*glyph.X - placement.geometry.Width) / 2
		placement.geometry.Y += (int32(rows)*glyph.Y - placement.geometry.Height) / 2
	}
	return placement, nil
}

func (term *Term) placeKittyImage(img *kittyImage, c *kittygraphics.Command, loader types.GraphicsRenderer) error {
	placement, err := term.kittyGeometry(img, c)
	if err != nil {
		return err
	}
	replacement := -1
	if img.id > 0 && placement.id > 0 {
		for index, existing := range term._kitty.placements {
			if existing.image == img && existing.id == placement.id {
				replacement = index
				break
			}
		}
	}
	if replacement < 0 && len(term._kitty.placements) >= kittyPlacements {
		return fmt.Errorf("ENOSPC:terminal image placement quota exceeded")
	}
	if img.asset == nil {
		img.asset, err = loader.LoadGraphicsImage(img.bitmap.PNG)
		if err != nil {
			return fmt.Errorf("EINVAL:cannot cache image: %w", err)
		}
	}
	if replacement >= 0 {
		term._kitty.placements[replacement] = placement
	} else {
		term._kitty.placements = append(term._kitty.placements, placement)
	}
	if c.Number("C") != 1 {
		for row := int32(0); row < min(placement.rows, term.size.Y); row++ {
			term.lineFeed(_LINEFEED_CURSOR_MOVED)
		}
		term._curPos.X = min(term.size.X-1, term._curPos.X+placement.columns)
	}
	term.renderer.TriggerRedraw()
	return nil
}

func (term *Term) kittyImagePlaced(img *kittyImage) bool {
	for _, p := range term._kitty.placements {
		if p.image == img {
			return true
		}
	}
	return false
}

func (term *Term) removeKittyImage(img *kittyImage) {
	state := term._kitty
	placements := state.placements[:0]
	for _, p := range state.placements {
		if p.image != img {
			placements = append(placements, p)
		}
	}
	clear(state.placements[len(placements):])
	state.placements = placements
	images := state.images[:0]
	for _, candidate := range state.images {
		if candidate != img {
			images = append(images, candidate)
		}
	}
	clear(state.images[len(images):])
	state.images = images
	state.bytes -= img.bitmap.Bytes
	if img.asset != nil {
		img.asset.Close()
	}
}

func (term *Term) clearKittyGraphics() {
	if term._kitty == nil {
		return
	}
	for _, img := range term._kitty.images {
		if img.asset != nil {
			img.asset.Close()
		}
	}
	term._kitty = nil
}

func (term *Term) ReleaseGraphics() {
	term._mutex.Lock()
	defer term._mutex.Unlock()
	term.clearKittyGraphics()
}

func (term *Term) kittyRows(alt bool) types.Screen {
	if alt {
		return term._altBuf
	}
	return append(append(types.Screen(nil), term._scrollBuf...), term._normBuf...)
}

func (term *Term) pruneKittyPlacements() {
	if term._kitty == nil {
		return
	}
	rows := make(map[*types.Row]bool)
	for _, screen := range []types.Screen{term.kittyRows(false), term._altBuf} {
		var collect func(types.Screen)
		collect = func(screen types.Screen) {
			for _, row := range screen {
				rows[row] = true
				collect(row.Hidden)
			}
		}
		collect(screen)
	}
	placements := term._kitty.placements[:0]
	for _, p := range term._kitty.placements {
		if rows[p.anchor] {
			placements = append(placements, p)
		}
	}
	clear(term._kitty.placements[len(placements):])
	term._kitty.placements = placements
	for _, img := range append([]*kittyImage(nil), term._kitty.images...) {
		if img.id == 0 && !term.kittyImagePlaced(img) {
			term.removeKittyImage(img)
		}
	}
}

func (term *Term) kittyRowOffset(p *kittyPlacement, screen types.Screen) (int32, bool) {
	if p.alt != term.IsAltBuf() || len(screen) == 0 {
		return 0, false
	}
	rows := term.kittyRows(p.alt)
	anchor, start := -1, -1
	for index, row := range rows {
		if row == p.anchor {
			anchor = index
		}
		if row == screen[0] {
			start = index
		}
	}
	if anchor < 0 || start < 0 {
		return 0, false
	}
	return int32(anchor - start), true
}

func (term *Term) kittyDrawPlacements(screen types.Screen) []kittyPlacement {
	if term._kitty == nil || len(term._kitty.placements) == 0 || len(screen) == 0 {
		return nil
	}
	rowIndices := make(map[*types.Row]int)
	for index, row := range term.kittyRows(term.IsAltBuf()) {
		rowIndices[row] = index
	}
	start, ok := rowIndices[screen[0]]
	if !ok {
		return nil
	}
	var placements []kittyPlacement
	for _, p := range term._kitty.placements {
		index, ok := rowIndices[p.anchor]
		y := int32(index - start)
		if !ok || p.alt != term.IsAltBuf() || y >= term.size.Y || y+p.rows <= 0 {
			continue
		}
		copyPlacement := *p
		copyPlacement.geometry.Row = y
		copyPlacement.geometry.ClipColumns, copyPlacement.geometry.ClipRows = term.size.X, term.size.Y
		placements = append(placements, copyPlacement)
	}
	sort.SliceStable(placements, func(i, j int) bool {
		if placements[i].z == placements[j].z {
			return placements[i].image.id < placements[j].image.id
		}
		return placements[i].z < placements[j].z
	})
	return placements
}

func (term *Term) drawKittyGraphics(placements []kittyPlacement, layer int) {
	for _, p := range placements {
		pLayer := 2
		if p.z < -1073741824 {
			pLayer = 0
		} else if p.z < 0 {
			pLayer = 1
		}
		if pLayer != layer {
			continue
		}
		p.image.asset.DrawGraphics(term.tile, p.geometry)
	}
}

func (term *Term) renderKittyCellBackgrounds(screen types.Screen) {
	for y, row := range screen {
		if y >= int(term.size.Y) {
			break
		}
		for x, cell := range row.Cells {
			if x >= int(term.size.X) {
				break
			}
			if cell.Sgr == nil || cell.HasDrawableElement() {
				continue
			}
			background := cell.Sgr.Bg
			if cell.Sgr.Bitwise.Is(types.SGR_INVERT) {
				background = cell.Sgr.Fg
			}
			if background != nil && background != types.SGR_COLOR_BACKGROUND {
				term.renderer.PrintCell(term.tile, &types.Cell{Char: ' ',
					Sgr: &types.Sgr{Fg: types.SGR_COLOR_FOREGROUND, Bg: background}},
					&types.XY{X: int32(x), Y: int32(y)})
			}
		}
	}
}

func (term *Term) kittyRenderCell(cell *types.Cell) *types.Cell {
	cell = term.textElementRenderCell(cell)
	if !term._kittyTextPass {
		return cell
	}
	copyCell := *cell
	copyCell.Sgr = cell.Sgr.Copy()
	if copyCell.Sgr.Bitwise.Is(types.SGR_INVERT) {
		copyCell.Sgr.Fg = copyCell.Sgr.Bg
		copyCell.Sgr.Bitwise.Unset(types.SGR_INVERT)
	}
	copyCell.Sgr.Bg = nil
	return &copyCell
}

func (term *Term) deleteKittyPlacements(c *kittygraphics.Command) {
	mode := c.String("d", "a")
	lower := strings.ToLower(mode)
	screen := *term.screen
	affected := make(map[*kittyImage]bool)
	placements := term._kitty.placements[:0]
	for _, p := range term._kitty.placements {
		row, exists := term.kittyRowOffset(p, screen)
		visible := exists && row < term.size.Y && row+p.rows > 0 && p.geometry.Column < term.size.X
		x, y := int32(c.Number("x"))-1, int32(c.Number("y"))-1
		if lower == "c" {
			x, y = term._curPos.X, term._curPos.Y
		}
		hitsX := x >= p.geometry.Column && x < p.geometry.Column+p.columns
		hitsY := y >= row && y < row+p.rows
		remove := false
		switch lower {
		case "a":
			remove = visible
		case "i":
			remove = p.image.id != 0 && int64(p.image.id) == c.Number("i") && (c.Number("p") == 0 || int64(p.id) == c.Number("p"))
		case "c", "p":
			remove = visible && hitsX && hitsY
		case "q":
			remove = visible && hitsX && hitsY && int64(p.z) == c.Number("z")
		case "x":
			remove = visible && hitsX
		case "y":
			remove = visible && hitsY
		case "z":
			remove = int64(p.z) == c.Number("z")
		case "r":
			remove = int64(p.image.id) >= c.Number("x") && int64(p.image.id) <= c.Number("y")
		}
		if remove {
			affected[p.image] = true
		} else {
			placements = append(placements, p)
		}
	}
	clear(term._kitty.placements[len(placements):])
	term._kitty.placements = placements
	for _, img := range append([]*kittyImage(nil), term._kitty.images...) {
		if lower == "i" && int64(img.id) == c.Number("i") && c.Number("p") == 0 {
			affected[img] = true
		}
		if lower == "r" && int64(img.id) >= c.Number("x") && int64(img.id) <= c.Number("y") {
			affected[img] = true
		}
		if (mode != lower && affected[img] || img.id == 0) && !term.kittyImagePlaced(img) {
			term.removeKittyImage(img)
		}
	}
	term.renderer.TriggerRedraw()
}

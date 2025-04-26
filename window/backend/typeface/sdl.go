package typeface

import (
	"fmt"
	"log"

	"github.com/flopp/go-findfont"
	"github.com/lmorg/mxtty/assets"
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type fontSdl struct {
	size *types.XY
	font *ttf.Font
}

func (f *fontSdl) Open(name string, size int) (err error) {
	if name != "" {
		err = f.openSystemTtf(name, size)
	}
	if name == "" || err != nil {
		f.font, err = f.openCompiledTtf(assets.TYPEFACE, size)
	}

	if err != nil {
		return err
	}

	f.font.SetHinting(ttf.HINTING_MONO)

	err = f.setSize()
	f.font.Close()
	return err
}

func (f *fontSdl) setSize() error {
	x, y, err := f.font.SizeUTF8("W")
	f.size = &types.XY{
		X: int32(x + config.Config.TypeFace.AdjustCellWidth),
		Y: int32(y + config.Config.TypeFace.AdjustCellHeight),
	}
	return err
}

func (f *fontSdl) openSystemTtf(name string, size int) error {
	path, err := findfont.Find(name)
	if err != nil {
		log.Printf("error in findfont.Find(): %s", err.Error())
		log.Println("defaulting to compiled font...")
	}

	f.font, err = ttf.OpenFont(path, size)
	if err != nil {
		return fmt.Errorf("error in ttf.OpenFont(): %s", err.Error())
	}

	return nil
}

func (f *fontSdl) openCompiledTtf(assetName string, size int) (*ttf.Font, error) {
	rwops, err := sdl.RWFromMem(assets.Get(assetName))
	if err != nil {
		return nil, fmt.Errorf("error in sdl.RWFromMem(): %s", err.Error())
	}

	font, err := ttf.OpenFontRW(rwops, 0, size)
	if err != nil {
		return nil, fmt.Errorf("error in ttf.OpenFontRW(): %s", err.Error())
	}
	return font, nil
}

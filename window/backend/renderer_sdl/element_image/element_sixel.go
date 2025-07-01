package element_image

import (
	"bytes"
	"fmt"
	"image"

	"github.com/lmorg/mxtty/types"
	"github.com/mattn/go-sixel"
	"golang.org/x/image/bmp"
)

func (el *ElementImage) fromSixel(apc *types.ApcSlice) error {
	el.escSeq = []byte(apc.Index(0))

	img, err := el.fromSixel_decodeSixel()
	if err != nil {
		return fmt.Errorf("unable to load image: %s", err.Error())
	}

	var b []byte
	buf := bytes.NewBuffer(b)

	err = bmp.Encode(buf, *img)
	if err != nil {
		return fmt.Errorf("unable to convert sixel to bitmap: %v", err)
	}

	el.bmp = buf.Bytes()
	return nil
}

func (el *ElementImage) fromSixel_decodeSixel() (*image.Image, error) {
	reader := bytes.NewReader(el.escSeq)
	var img image.Image
	err := sixel.NewDecoder(reader).Decode(&img)

	return &img, err
}

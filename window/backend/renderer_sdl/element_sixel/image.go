package elementSixel

import (
	"bytes"
	"fmt"
	"image"

	"github.com/mattn/go-sixel"
	"golang.org/x/image/bmp"
)

func (el *ElementImage) decode() error {
	img, err := el.decodeSixel()
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

func (el *ElementImage) decodeSixel() (*image.Image, error) {
	reader := bytes.NewReader(el.escSeq)
	var img image.Image
	err := sixel.NewDecoder(reader).Decode(&img)

	return &img, err
}

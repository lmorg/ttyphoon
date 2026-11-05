package kittygraphics

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"strconv"
	"strings"
)

const (
	MaxChunkBytes = 4096
	MaxImageBytes = 64 << 20
	MaxPixels     = MaxImageBytes / 4
)

// Command holds the first chunk's controls and the completed transfer.
type Command struct {
	Controls map[string]string
	Data     []byte
}

func (c Command) String(key, fallback string) string {
	if value, ok := c.Controls[key]; ok {
		return value
	}
	return fallback
}

func (c Command) Number(key string) int64 {
	value, _ := strconv.ParseInt(c.Controls[key], 10, 64)
	return value
}

type Upload struct {
	pending *Command
}

func (u *Upload) Reset() { u.pending = nil }

func (u *Upload) abort(c *Command, err error) (*Command, error) {
	if u.pending != nil {
		c = u.pending
	}
	u.Reset()
	return c, err
}

// Push accepts the bytes after ESC_G, without the terminating ST.
// A nil result denotes an incomplete, validated chunked transfer.
func (u *Upload) Push(text string) (*Command, error) {
	header, payload, _ := strings.Cut(text, ";")
	c := &Command{Controls: make(map[string]string)}
	for _, field := range strings.Split(header, ",") {
		if field == "" {
			continue
		}
		key, value, ok := strings.Cut(field, "=")
		if !ok || len(key) != 1 || value == "" || c.Controls[key] != "" {
			return u.abort(c, fmt.Errorf("EINVAL:invalid graphics control"))
		}
		c.Controls[key] = value
	}
	if err := validate(c); err != nil {
		return u.abort(c, err)
	}
	if c.String("a", "") == "d" {
		u.Reset()
		if payload != "" {
			return c, fmt.Errorf("EINVAL:delete has image data")
		}
		return c, nil
	}
	more := c.Number("m") == 1
	if u.pending != nil {
		for key := range c.Controls {
			if key != "m" && key != "q" {
				previous := u.pending
				u.Reset()
				return previous, fmt.Errorf("EINVAL:interrupted graphics upload")
			}
		}
		if _, ok := c.Controls["m"]; !ok {
			previous := u.pending
			u.Reset()
			return previous, fmt.Errorf("EINVAL:missing continuation flag")
		}
		if quiet, ok := c.Controls["q"]; ok {
			u.pending.Controls["q"] = quiet
		}
		c = u.pending
	}
	if len(payload) > MaxChunkBytes {
		u.Reset()
		return c, fmt.Errorf("E2BIG:graphics chunk exceeds 4096 bytes")
	}
	data, err := base64.StdEncoding.Strict().DecodeString(payload)
	if err != nil {
		u.Reset()
		return c, fmt.Errorf("EINVAL:invalid base64 image data")
	}
	if len(c.Data)+len(data) > MaxImageBytes {
		u.Reset()
		return c, fmt.Errorf("E2BIG:image transfer exceeds quota")
	}
	c.Data = append(c.Data, data...)
	if more {
		if len(payload)%4 != 0 || strings.Contains(payload, "=") {
			u.Reset()
			return c, fmt.Errorf("EINVAL:invalid intermediate chunk")
		}
		u.pending = c
		return nil, nil
	}
	u.Reset()
	return c, nil
}

func validate(c *Command) error {
	for key, value := range c.Controls {
		switch key {
		case "a":
			if !strings.Contains("tTpqd", value) || len(value) != 1 {
				return fmt.Errorf("ENOTSUP:unsupported graphics action")
			}
		case "t":
			if value != "d" {
				return fmt.Errorf("ENOTSUP:only direct image transmission is supported")
			}
		case "o":
			if value != "z" {
				return fmt.Errorf("ENOTSUP:unsupported image compression")
			}
		case "d":
			if len(value) != 1 || !strings.Contains("aAiIcCpPqQxXyYzZrR", value) {
				return fmt.Errorf("ENOTSUP:unsupported image deletion")
			}
		case "z", "i", "p", "s", "v", "f", "m", "q", "x", "y", "w", "h", "c", "r", "X", "Y", "C":
			number, err := strconv.ParseInt(value, 10, 64)
			if err != nil || (key == "z" && (number < -1<<31 || number > 1<<31-1)) ||
				(key != "z" && (number < 0 || number > 1<<32-1)) {
				return fmt.Errorf("EINVAL:invalid numeric graphics control %s", key)
			}
			if (key == "m" || key == "C") && number > 1 || key == "q" && number > 2 {
				return fmt.Errorf("EINVAL:invalid graphics flag %s", key)
			}
			if key == "f" && number != 24 && number != 32 && number != 100 {
				return fmt.Errorf("ENOTSUP:unsupported image format")
			}
		default:
			return fmt.Errorf("ENOTSUP:unsupported graphics control %s", key)
		}
	}
	return nil
}

type Bitmap struct {
	PNG    []byte
	Width  int
	Height int
	Bytes  int
}

func Decode(c *Command) (*Bitmap, error) {
	data := c.Data
	if c.String("o", "") == "z" {
		reader, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("EINVAL:invalid zlib image data")
		}
		data, err = io.ReadAll(io.LimitReader(reader, MaxImageBytes+1))
		closeErr := reader.Close()
		if err != nil || closeErr != nil {
			return nil, fmt.Errorf("EINVAL:cannot decompress image")
		}
		if len(data) > MaxImageBytes {
			return nil, fmt.Errorf("E2BIG:decompressed image exceeds quota")
		}
	}
	width, height := c.Number("s"), c.Number("v")
	var img image.Image
	if c.String("f", "32") == "100" {
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("EINVAL:invalid PNG header")
		}
		width, height = int64(cfg.Width), int64(cfg.Height)
		if !validDimensions(width, height) {
			return nil, fmt.Errorf("E2BIG:invalid or oversized PNG dimensions")
		}
		img, err = png.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("EINVAL:invalid PNG image")
		}
	} else {
		if !validDimensions(width, height) {
			return nil, fmt.Errorf("E2BIG:invalid or oversized image dimensions")
		}
		stride := int64(4)
		if c.String("f", "32") == "24" {
			stride = 3
		}
		if int64(len(data)) != width*height*stride {
			return nil, fmt.Errorf("EINVAL:pixel data does not match dimensions")
		}
		rgba := image.NewNRGBA(image.Rect(0, 0, int(width), int(height)))
		for src, dst := 0, 0; src < len(data); src, dst = src+int(stride), dst+4 {
			copy(rgba.Pix[dst:dst+3], data[src:src+3])
			rgba.Pix[dst+3] = 255
			if stride == 4 {
				rgba.Pix[dst+3] = data[src+3]
			}
		}
		img = rgba
	}
	var output bytes.Buffer
	if err := png.Encode(&output, img); err != nil {
		return nil, fmt.Errorf("EINVAL:cannot encode image: %w", err)
	}
	return &Bitmap{PNG: output.Bytes(), Width: int(width), Height: int(height),
		Bytes: int(width*height*4) + output.Len()}, nil
}

func validDimensions(width, height int64) bool {
	return width > 0 && height > 0 && width <= MaxPixels && height <= MaxPixels && width*height <= MaxPixels
}

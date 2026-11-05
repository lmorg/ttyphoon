package kittygraphics

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestDecodeFormatsAndCompression(t *testing.T) {
	var pngData bytes.Buffer
	pixel := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	pixel.SetNRGBA(0, 0, color.NRGBA{R: 40, G: 80, B: 120, A: 128})
	if err := png.Encode(&pngData, pixel); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		format string
		data   []byte
		alpha  uint8
	}{
		{"24", []byte{40, 80, 120}, 255},
		{"32", []byte{40, 80, 120, 128}, 128},
		{"100", pngData.Bytes(), 128},
	} {
		for _, compressed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/compressed=%v", tc.format, compressed), func(t *testing.T) {
				data := tc.data
				controls := map[string]string{"s": "1", "v": "1", "f": tc.format}
				if compressed {
					var zipped bytes.Buffer
					writer := zlib.NewWriter(&zipped)
					if _, err := writer.Write(data); err != nil {
						t.Fatal(err)
					}
					if err := writer.Close(); err != nil {
						t.Fatal(err)
					}
					data = zipped.Bytes()
					controls["o"] = "z"
				}
				bitmap, err := Decode(&Command{Controls: controls, Data: data})
				if err != nil {
					t.Fatal(err)
				}
				img, err := png.Decode(bytes.NewReader(bitmap.PNG))
				if err != nil {
					t.Fatal(err)
				}
				got := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA)
				if got != (color.NRGBA{R: 40, G: 80, B: 120, A: tc.alpha}) {
					t.Fatalf("pixel changed: %#v", got)
				}
				if bitmap.Width != 1 || bitmap.Height != 1 || bitmap.Bytes != 4+len(bitmap.PNG) {
					t.Fatal("wrong dimensions or quota accounting")
				}
			})
		}
	}
}

func TestChunkedUpload(t *testing.T) {
	var upload Upload
	first, err := upload.Push("a=T,i=4294967295,p=2,f=24,s=2,v=1,m=1;AQID")
	if err != nil || first != nil {
		t.Fatalf("premature upload: %v %v", first, err)
	}
	last, err := upload.Push("m=0,q=2;BAUG")
	if err != nil || last == nil {
		t.Fatalf("incomplete upload: %v", err)
	}
	if !bytes.Equal(last.Data, []byte{1, 2, 3, 4, 5, 6}) || last.Number("q") != 2 || last.Number("i") != 4294967295 {
		t.Fatal("lost first-chunk metadata or data")
	}
	if _, err := Decode(last); err != nil {
		t.Fatal(err)
	}
	if upload.pending != nil {
		t.Fatal("upload not released")
	}
}

func TestInvalidControlsAndChunks(t *testing.T) {
	for _, input := range []string{
		"a=f;iA==", "a=t,t=f;AAAA", "a=t,f=8;AAAA", "U=1;AAAA",
		"i=-1;AAAA", "i=4294967296;AAAA", "z=-2147483649;AAAA",
		"q=3;AAAA", "m=2;AAAA", "C=2;AAAA", "s=abc;AAAA",
		"i=1,i=2;AAAA", "garbage;AAAA", "s=1;!!!!", "m=1;AA==",
		"m=1;" + strings.Repeat("A", MaxChunkBytes+4), "a=d;AAAA",
	} {
		t.Run(input[:min(len(input), 30)], func(t *testing.T) {
			var upload Upload
			if _, err := upload.Push(input); err == nil {
				t.Fatal("accepted invalid controls or data")
			}
			if upload.pending != nil {
				t.Fatal("retained invalid upload")
			}
		})
	}
	var upload Upload
	upload.Push("s=2,v=1,f=24,m=1;AAAA")
	c, err := upload.Push("a=p,i=7")
	if err == nil || c.Number("s") != 2 || upload.pending != nil {
		t.Fatal("interrupted transfer did not fail and release data")
	}
	upload.Push("s=2,v=1,f=24,m=1;AAAA")
	if _, err := upload.Push("a=d,d=A"); err != nil || upload.pending != nil {
		t.Fatal("deletion did not abort upload")
	}
}

func TestDecodeRejectsBadAndOversizedImages(t *testing.T) {
	for _, command := range []*Command{
		{Controls: map[string]string{"s": "0", "v": "1", "f": "24"}, Data: []byte{0, 0, 0}},
		{Controls: map[string]string{"s": "4294967295", "v": "4294967295"}, Data: []byte{0}},
		{Controls: map[string]string{"s": "1", "v": "1", "f": "24"}, Data: []byte{0, 0}},
		{Controls: map[string]string{"f": "100"}, Data: []byte("not png")},
		{Controls: map[string]string{"o": "z"}, Data: []byte("not zlib")},
	} {
		if _, err := Decode(command); err == nil {
			t.Fatal("accepted invalid or oversized image")
		}
	}
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	block := make([]byte, 1<<20)
	for n := 0; n < 65; n++ {
		if _, err := writer.Write(block); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(&Command{Controls: map[string]string{"o": "z"}, Data: compressed.Bytes()}); err == nil || !strings.HasPrefix(err.Error(), "E2BIG:") {
		t.Fatal("decompression quota not enforced")
	}

	var upload Upload
	upload.pending = &Command{Controls: map[string]string{"f": "24"}, Data: make([]byte, MaxImageBytes)}
	if _, err := upload.Push("m=0;" + base64.StdEncoding.EncodeToString([]byte{1})); err == nil || upload.pending != nil {
		t.Fatal("transfer quota not enforced")
	}
}

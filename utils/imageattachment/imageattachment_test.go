package imageattachment

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

func TestValidateAndStoreImageAttachments(t *testing.T) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	attachment := types.ImageAttachment{MIMEType: "image/png", Base64: base64.StdEncoding.EncodeToString(buffer.Bytes())}
	if err := Validate([]types.ImageAttachment{attachment}); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	stored, err := storeInDirectory([]types.ImageAttachment{attachment}, directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 || filepath.Dir(stored[0].Path) != directory || stored[0].Base64 != attachment.Base64 {
		t.Fatalf("unexpected stored attachment: %+v", stored)
	}
	data, err := os.ReadFile(stored[0].Path)
	if err != nil || !bytes.Equal(data, buffer.Bytes()) {
		t.Fatalf("stored image differs: %v", err)
	}
	for _, invalid := range [][]types.ImageAttachment{
		{{MIMEType: "image/svg+xml", Base64: attachment.Base64}},
		{{MIMEType: "image/jpeg", Base64: attachment.Base64}},
		{{MIMEType: "image/png", Base64: "invalid"}},
		{{MIMEType: "image/png", Base64: ""}},
		{{MIMEType: "image/png", Base64: base64.StdEncoding.EncodeToString(make([]byte, MaxBytes+1))}},
		make([]types.ImageAttachment, MaxImages+1),
	} {
		if err := Validate(invalid); err == nil {
			t.Fatal("accepted invalid attachments")
		}
	}
}

func TestLoadSavedImagesIsBoundedAndRestrictedToAttachmentDirectory(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	imageData := base64.StdEncoding.EncodeToString(buffer.Bytes())
	stored, err := Store([]types.ImageAttachment{{MIMEType: "image/png", Base64: imageData}})
	if err != nil {
		t.Fatal(err)
	}
	reference := types.ImageAttachment{MIMEType: stored[0].MIMEType, Path: stored[0].Path}
	loaded, err := Load([]types.ImageAttachment{reference})
	if err != nil || len(loaded) != 1 || loaded[0].Base64 != imageData {
		t.Fatalf("could not reload image bytes: %+v, %v", loaded, err)
	}
	outside := filepath.Join(t.TempDir(), "outside.png")
	if err := os.WriteFile(outside, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load([]types.ImageAttachment{{MIMEType: "image/png", Path: outside}}); err == nil {
		t.Fatal("loaded a file outside the attachment directory")
	}
	if err := os.Remove(reference.Path); err != nil {
		t.Fatal(err)
	}
	if _, err := Load([]types.ImageAttachment{reference}); err == nil {
		t.Fatal("loaded a missing attachment")
	}
}

package imageattachment

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/file"
	_ "golang.org/x/image/webp"
)

const MaxImages = 6
const MaxBytes = 8 * 1024 * 1024

var formats = map[string]string{
	"image/png": "png", "image/jpeg": "jpeg", "image/gif": "gif", "image/webp": "webp",
}

func Validate(images []types.ImageAttachment) error {
	if len(images) > MaxImages {
		return fmt.Errorf("attach at most %d images", MaxImages)
	}
	for _, attachment := range images {
		format, ok := formats[attachment.MIMEType]
		if !ok {
			return fmt.Errorf("unsupported image type %q", attachment.MIMEType)
		}
		if len(attachment.Base64) > base64.StdEncoding.EncodedLen(MaxBytes) {
			return fmt.Errorf("each image must be at most 8 MiB")
		}
		data, err := base64.StdEncoding.DecodeString(attachment.Base64)
		if err != nil || len(data) == 0 || len(data) > MaxBytes {
			return fmt.Errorf("invalid image data or image exceeds 8 MiB")
		}
		config, decodedFormat, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || decodedFormat != format || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 40_000_000 {
			return fmt.Errorf("invalid image format or image exceeds 40 megapixels")
		}
	}
	return nil
}

func Store(images []types.ImageAttachment) ([]types.ImageAttachment, error) {
	if len(images) == 0 {
		return nil, nil
	}
	if err := Validate(images); err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return storeInDirectory(images, filepath.Join(file.DocumentsDir(home), ".images"))
}

func Load(images []types.ImageAttachment) ([]types.ImageAttachment, error) {
	if len(images) == 0 {
		return nil, nil
	}
	if len(images) > MaxImages {
		return nil, fmt.Errorf("attach at most %d images", MaxImages)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	directory, err := filepath.EvalSymlinks(filepath.Join(file.DocumentsDir(home), ".images"))
	if err != nil {
		return nil, err
	}
	loaded := make([]types.ImageAttachment, 0, len(images))
	for _, attachment := range images {
		path, err := filepath.EvalSymlinks(attachment.Path)
		if err != nil || filepath.Dir(path) != directory {
			return nil, fmt.Errorf("saved image is unavailable or outside the attachment directory")
		}
		imageFile, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		info, statErr := imageFile.Stat()
		if statErr != nil || !info.Mode().IsRegular() || info.Size() > MaxBytes {
			imageFile.Close()
			return nil, fmt.Errorf("saved image is invalid or exceeds 8 MiB")
		}
		data, readErr := io.ReadAll(io.LimitReader(imageFile, MaxBytes+1))
		closeErr := imageFile.Close()
		if readErr != nil || closeErr != nil || len(data) > MaxBytes {
			return nil, fmt.Errorf("could not read saved image")
		}
		attachment.Base64 = base64.StdEncoding.EncodeToString(data)
		loaded = append(loaded, attachment)
	}
	if err := Validate(loaded); err != nil {
		return nil, err
	}
	return loaded, nil
}

func storeInDirectory(images []types.ImageAttachment, directory string) ([]types.ImageAttachment, error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}
	stored := make([]types.ImageAttachment, 0, len(images))
	for _, attachment := range images {
		data, err := base64.StdEncoding.DecodeString(attachment.Base64)
		if err != nil {
			return nil, err
		}
		imageFile, err := os.CreateTemp(directory, "inputbox-*."+formats[attachment.MIMEType])
		if err != nil {
			return nil, err
		}
		_, writeErr := imageFile.Write(data)
		closeErr := imageFile.Close()
		if writeErr != nil || closeErr != nil {
			_ = os.Remove(imageFile.Name())
			return nil, fmt.Errorf("save image: %v; close image: %v", writeErr, closeErr)
		}
		attachment.Path = imageFile.Name()
		stored = append(stored, attachment)
	}
	return stored, nil
}

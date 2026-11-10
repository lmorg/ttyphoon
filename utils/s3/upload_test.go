package s3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lmorg/ttyphoon/config"
)

func settings() config.S3Config {
	return config.S3Config{Region: "us-east-1", AccessKeyID: "test-key", SecretAccessKey: "test-secret",
		SessionToken: "test-session", Bucket: "test-bucket", Prefix: "clipboard/images",
		PublicBaseURL: "https://cdn.example.com/assets"}
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func TestUploadPNGSignsAndPublishesConfiguredPath(t *testing.T) {
	data := testPNG(t)
	var uploadedPath string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uploadedPath = r.URL.Path
		body, err := io.ReadAll(r.Body)
		if err != nil || !bytes.Equal(body, data) {
			t.Error("upload bytes changed")
		}
		hash := sha256.Sum256(data)
		if r.Method != http.MethodPut || r.Header.Get("Content-Type") != "image/png" ||
			r.Header.Get("X-Amz-Content-Sha256") != hex.EncodeToString(hash[:]) ||
			r.Header.Get("X-Amz-Security-Token") != "test-session" ||
			!strings.Contains(r.Header.Get("Authorization"), "Credential=test-key/") ||
			!strings.Contains(r.Header.Get("Authorization"), "/us-east-1/s3/aws4_request") {
			t.Error("missing S3 signature or upload headers")
		}
		if r.Header.Get("X-Amz-Acl") != "" {
			t.Error("upload must not force a public ACL")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	cfg := settings()
	cfg.Endpoint = server.URL + "/s3"
	uploader, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	uploader.client.Transport = server.Client().Transport
	address, err := uploader.UploadPNG(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	prefix := "/s3/test-bucket/clipboard/images/"
	if !strings.HasPrefix(uploadedPath, prefix) {
		t.Fatalf("upload path = %q", uploadedPath)
	}
	filename := strings.TrimPrefix(uploadedPath, prefix)
	if _, err := uuid.Parse(strings.TrimSuffix(filename, ".png")); err != nil {
		t.Fatalf("filename is not a UUID PNG: %s", filename)
	}
	if address != "https://cdn.example.com/assets/clipboard/images/"+filename {
		t.Fatalf("public URL = %q", address)
	}
}

func TestUploadErrorsAndRedirects(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusTemporaryRedirect, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Location", "/elsewhere")
				w.WriteHeader(status)
			}))
			defer server.Close()
			cfg := settings()
			cfg.Endpoint = server.URL
			uploader, err := New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			uploader.client.Transport = server.Client().Transport
			if _, err := uploader.UploadPNG(context.Background(), testPNG(t)); err == nil {
				t.Fatal("expected upload error")
			}
			if calls != 1 {
				t.Fatalf("followed upload redirect: %d calls", calls)
			}
		})
	}
}

func TestSettingsValidationAndLiteralValues(t *testing.T) {
	t.Setenv("TTYPHOON_TEST_S3_KEY", "expanded-key")
	cfg := settings()
	cfg.AccessKeyID = "$TTYPHOON_TEST_S3_KEY"
	cfg.SecretAccessKey = "${TTYPHOON_TEST_S3_KEY}"
	cfg.SessionToken = "$TTYPHOON_TEST_S3_KEY"
	cfg.Prefix = "$TTYPHOON_TEST_S3_KEY"
	uploader, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if uploader.creds.AccessKeyID != cfg.AccessKeyID || uploader.creds.SecretAccessKey != cfg.SecretAccessKey ||
		uploader.creds.SessionToken != cfg.SessionToken || uploader.prefix != cfg.Prefix {
		t.Fatal("constructor expanded literal configuration values")
	}
	for _, change := range []func(*config.S3Config){
		func(c *config.S3Config) { c.SecretAccessKey = "" },
		func(c *config.S3Config) { c.Bucket = "../bucket" },
		func(c *config.S3Config) { c.Prefix = "../path" },
		func(c *config.S3Config) { c.PublicBaseURL = "http://example.com" },
		func(c *config.S3Config) { c.Endpoint = "https://user:password@example.com" },
		func(c *config.S3Config) { c.PublicBaseURL = "https://example.com?token=secret" },
	} {
		cfg := settings()
		change(&cfg)
		if _, err := New(cfg); err == nil {
			t.Fatal("accepted invalid upload configuration")
		}
	}
}

func TestInvalidPNGDoesNotUpload(t *testing.T) {
	uploader, err := New(settings())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := uploader.UploadPNG(context.Background(), []byte("<svg/>")); err == nil {
		t.Fatal("accepted non-PNG upload")
	}
}

func TestFromConfigOptIn(t *testing.T) {
	previous := config.Config.Social.S3
	t.Cleanup(func() { config.Config.Social.S3 = previous })
	config.Config.Social.S3 = nil
	if uploader, err := FromConfig(); err != nil || uploader != nil {
		t.Fatal("absent S3 settings should disable uploads")
	}
	config.Config.Social.S3 = &config.S3Config{}
	if _, err := FromConfig(); err == nil {
		t.Fatal("incomplete S3 settings should report an error")
	}
	cfg := settings()
	config.Config.Social.S3 = &cfg
	if uploader, err := FromConfig(); err != nil || uploader == nil {
		t.Fatalf("configured S3 upload not enabled: %v", err)
	}
}

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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/google/uuid"
	"github.com/lmorg/ttyphoon/app"
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
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
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
		func(c *config.S3Config) { c.Region = " " },
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

func TestScopedEnvironmentSignsUploadsAndPreservesIsolation(t *testing.T) {
	for key, value := range map[string]string{
		"AWS_ACCESS_KEY_ID": "host-key", "AWS_SECRET_ACCESS_KEY": "host-secret",
		"AWS_SESSION_TOKEN": "host-token", "AWS_REGION": "host-region",
		"AWS_DEFAULT_REGION": "host-default",
	} {
		t.Setenv(key, value)
	}
	first, second := t.TempDir(), t.TempDir()
	previousDirs := xdg.ConfigDirs
	xdg.ConfigDirs = []string{first, second}
	t.Cleanup(func() { xdg.ConfigDirs = previousDirs })
	write := func(path, contents string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(first, app.DirName, "s3.vars"), "AWS_ACCESS_KEY_ID=first-key\nAWS_SECRET_ACCESS_KEY=file-secret\nAWS_REGION=first-region\n")
	write(filepath.Join(second, app.DirName, "s3.vars"), "AWS_ACCESS_KEY_ID=second-key\nAWS_REGION=file-region\nAWS_SESSION_TOKEN=file-token\n")
	last := filepath.Join(t.TempDir(), "last.vars")
	write(last, "AWS_ACCESS_KEY_ID=last-key\nAWS_SECRET_ACCESS_KEY=last-secret\n")
	requests := make(chan http.Header, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	cfg := config.S3Config{
		EnvFiles: []string{"s3.vars", last}, Endpoint: server.URL, Bucket: "test-bucket",
		PublicBaseURL: "https://cdn.example.com",
	}
	uploader, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	uploader.client.Transport = server.Client().Transport
	if _, err := uploader.UploadPNG(context.Background(), testPNG(t)); err != nil {
		t.Fatal(err)
	}
	headers := <-requests
	if !strings.Contains(headers.Get("Authorization"), "Credential=last-key/") ||
		!strings.Contains(headers.Get("Authorization"), "/file-region/s3/aws4_request") ||
		headers.Get("X-Amz-Security-Token") != "file-token" {
		t.Fatal("signed upload did not use the scoped credentials and region")
	}
	if uploader.creds.SecretAccessKey != "last-secret" || os.Getenv("AWS_ACCESS_KEY_ID") != "host-key" {
		t.Fatal("scoped S3 credentials were not isolated from the host")
	}
	cfg.Region, cfg.AccessKeyID, cfg.SecretAccessKey, cfg.SessionToken = "literal-region", "literal-key", "literal-secret", "literal-token"
	literal, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if literal.region != "literal-region" || literal.creds.AccessKeyID != "literal-key" ||
		literal.creds.SecretAccessKey != "literal-secret" || literal.creds.SessionToken != "literal-token" {
		t.Fatal("literal S3 fields did not override environment files")
	}
	if err := os.WriteFile(last, []byte("AWS_ACCESS_KEY_ID=new-key\nAWS_SECRET_ACCESS_KEY=new-secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg.Region, cfg.AccessKeyID, cfg.SecretAccessKey, cfg.SessionToken = "", "", "", ""
	reloaded, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.creds.AccessKeyID != "new-key" || uploader.creds.AccessKeyID != "last-key" {
		t.Fatal("new uploader did not load updated files independently")
	}
}

func TestS3EnvironmentFallbackAndEmptyValues(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "host-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "host-secret")
	t.Setenv("AWS_SESSION_TOKEN", "host-token")
	t.Setenv("AWS_REGION", "")
	if err := os.Unsetenv("AWS_REGION"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_DEFAULT_REGION", "host-default")
	cfg := config.S3Config{Bucket: "bucket", PublicBaseURL: "https://cdn.example.com"}
	uploader, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if uploader.region != "host-default" || uploader.creds.AccessKeyID != "host-key" ||
		uploader.creds.SecretAccessKey != "host-secret" || uploader.creds.SessionToken != "host-token" {
		t.Fatal("S3 did not use host environment fallbacks")
	}
	path := filepath.Join(t.TempDir(), "s3.vars")
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cfg.EnvFiles = []string{path}
	write("AWS_REGION=file-region\nAWS_ACCESS_KEY_ID=\n")
	if _, err := New(cfg); err == nil || !strings.Contains(err.Error(), "AccessKeyID") {
		t.Fatalf("empty scoped key fell through to host: %v", err)
	}
	cfg.AccessKeyID = "literal-key"
	if uploader, err := New(cfg); err != nil || uploader.creds.AccessKeyID != "literal-key" {
		t.Fatalf("literal credential did not override empty scoped value: %v", err)
	}
	cfg.AccessKeyID = ""
	write("AWS_SECRET_ACCESS_KEY=\n")
	if _, err := New(cfg); err == nil || !strings.Contains(err.Error(), "SecretAccessKey") {
		t.Fatalf("empty scoped secret fell through to host: %v", err)
	}
	write("AWS_REGION=\nAWS_DEFAULT_REGION=file-default\n")
	if _, err := New(cfg); err == nil || !strings.Contains(err.Error(), "Region") {
		t.Fatalf("empty scoped region fell through to default: %v", err)
	}
	write("AWS_SESSION_TOKEN=\n")
	uploader, err = New(cfg)
	if err != nil || uploader.creds.SessionToken != "" || uploader.region != "host-default" {
		t.Fatalf("empty scoped token or default region not preserved: %v", err)
	}
	write("AWS_DEFAULT_REGION=file-default\n")
	uploader, err = New(cfg)
	if err != nil || uploader.region != "file-default" {
		t.Fatalf("scoped default region did not override its host value: %v", err)
	}
	t.Setenv("AWS_REGION", "host-primary")
	write("AWS_DEFAULT_REGION=file-default\n")
	uploader, err = New(cfg)
	if err != nil || uploader.region != "host-primary" {
		t.Fatalf("primary region was not preferred over fallback: %v", err)
	}
	write("AWS_SECRET_ACCESS_KEY=secret-value\n!INVALID=ignored\n")
	if _, err := New(cfg); err == nil || strings.Contains(err.Error(), "secret-value") {
		t.Fatalf("malformed environment file was not safely rejected: %v", err)
	}
	cfg.EnvFiles = []string{filepath.Join(t.TempDir(), "missing.vars")}
	if _, err := New(cfg); err == nil || !strings.Contains(err.Error(), "resolve Social/S3 environment") {
		t.Fatalf("missing S3 environment file was not reported: %v", err)
	}
	literal := settings()
	literal.EnvFiles = cfg.EnvFiles
	if _, err := New(literal); err == nil {
		t.Fatal("configured missing file was ignored for literal credentials")
	}
}

func TestS3DoesNotUseAWSProfileCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials")
	if err := os.WriteFile(path, []byte("[default]\naws_access_key_id=profile-key\naws_secret_access_key=profile-secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", path)
	t.Setenv("AWS_PROFILE", "default")
	for _, name := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"} {
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.S3Config{Region: "us-east-1", Bucket: "bucket", PublicBaseURL: "https://cdn.example.com"}
	if _, err := New(cfg); err == nil || !strings.Contains(err.Error(), "AccessKeyID") {
		t.Fatalf("S3 unexpectedly used AWS profile credentials: %v", err)
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
	t.Setenv("AWS_ACCESS_KEY_ID", "host-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "host-secret")
	t.Setenv("AWS_REGION", "host-region")
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

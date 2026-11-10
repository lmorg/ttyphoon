package s3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/png"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/google/uuid"
	"github.com/lmorg/ttyphoon/config"
)

type Uploader struct {
	endpoint *url.URL
	public   *url.URL
	region   string
	prefix   string
	creds    aws.Credentials
	client   *http.Client
}

// FromConfig returns nil when Social/S3 is absent. A present but incomplete
// configuration is an error, never an implicit opt-out.
func FromConfig() (*Uploader, error) {
	if config.Config.Social.S3 == nil {
		return nil, nil
	}
	return New(*config.Config.Social.S3)
}

func New(settings config.S3Config) (*Uploader, error) {
	for _, field := range []struct{ name, value string }{
		{"Region", settings.Region}, {"AccessKeyID", settings.AccessKeyID},
		{"SecretAccessKey", settings.SecretAccessKey}, {"Bucket", settings.Bucket},
		{"PublicBaseURL", settings.PublicBaseURL},
	} {
		if strings.TrimSpace(field.value) == "" {
			return nil, fmt.Errorf("Social/S3/%s is required", field.name)
		}
	}
	if strings.ContainsAny(settings.Bucket, `/\?#`) || settings.Bucket == "." || settings.Bucket == ".." {
		return nil, fmt.Errorf("Social/S3/Bucket must be a bucket name, not a path")
	}
	prefix := strings.Trim(settings.Prefix, "/")
	for _, segment := range strings.Split(prefix, "/") {
		if segment == "." || segment == ".." || strings.ContainsAny(segment, "\\\x00\r\n") {
			return nil, fmt.Errorf("Social/S3/Prefix contains an invalid path segment")
		}
	}
	parseURL := func(raw, field string) (*url.URL, error) {
		u, err := url.Parse(raw)
		if err != nil || u == nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return nil, fmt.Errorf("Social/S3/%s must be an HTTPS URL without credentials, query, or fragment", field)
		}
		return u, nil
	}
	public, err := parseURL(settings.PublicBaseURL, "PublicBaseURL")
	if err != nil {
		return nil, err
	}
	endpoint := settings.Endpoint
	if endpoint == "" {
		if strings.ContainsAny(settings.Region, "/\\:?# \t\r\n") {
			return nil, fmt.Errorf("Social/S3/Region is invalid")
		}
		endpoint = "https://s3." + settings.Region + ".amazonaws.com"
	}
	target, err := parseURL(endpoint, "Endpoint")
	if err != nil {
		return nil, err
	}
	target = target.JoinPath(settings.Bucket)
	return &Uploader{
		endpoint: target, public: public, region: settings.Region, prefix: prefix,
		creds: aws.Credentials{AccessKeyID: settings.AccessKeyID, SecretAccessKey: settings.SecretAccessKey, SessionToken: settings.SessionToken},
		client: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}},
	}, nil
}

// UploadPNG writes a private object; bucket/CDN policy controls public reading.
// PublicBaseURL identifies the bucket root; Prefix is appended to both URLs.
func (u *Uploader) UploadPNG(ctx context.Context, data []byte) (string, error) {
	if len(data) > 32<<20 {
		return "", fmt.Errorf("S3 image exceeds 32 MiB")
	}
	size, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "png" || size.Width <= 0 || size.Height <= 0 ||
		size.Width > 16384 || size.Height > 16384 || int64(size.Width)*int64(size.Height) > 16*1024*1024 {
		return "", fmt.Errorf("S3 upload requires a PNG within clipboard image limits")
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("generate S3 image name: %w", err)
	}
	key := id.String() + ".png"
	target := u.endpoint.JoinPath(u.prefix, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, target.String(), bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("create S3 upload request")
	}
	req.Header.Set("Content-Type", "image/png")
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", digest)
	if err := v4.NewSigner().SignHTTP(ctx, u.creds, req, digest, "s3", u.region, time.Now(), func(options *v4.SignerOptions) {
		options.DisableURIPathEscaping = true
	}); err != nil {
		return "", fmt.Errorf("sign S3 upload: %w", err)
	}
	response, err := u.client.Do(req)
	if err != nil {
		// Avoid disclosing endpoint URLs or credentials in clipboard warnings.
		return "", fmt.Errorf("S3 image upload failed: connection error, timeout, or cancellation")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("S3 image upload failed (HTTP %d)", response.StatusCode)
	}
	return u.public.JoinPath(u.prefix, key).String(), nil
}

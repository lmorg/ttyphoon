package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSocialS3YAML(t *testing.T) {
	var decoded configT
	decoder := yaml.NewDecoder(strings.NewReader(`
Social:
  S3:
    Endpoint: https://s3.example.com
    Region: us-east-1
    AccessKeyID: $S3_KEY
    SecretAccessKey: $S3_SECRET
    SessionToken: $S3_TOKEN
    Bucket: images
    Prefix: clipboard
    PublicBaseURL: https://cdn.example.com
`))
	decoder.KnownFields(true)
	if err := decoder.Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	want := S3Config{
		Endpoint: "https://s3.example.com", Region: "us-east-1",
		AccessKeyID: "$S3_KEY", SecretAccessKey: "$S3_SECRET", SessionToken: "$S3_TOKEN",
		Bucket: "images", Prefix: "clipboard", PublicBaseURL: "https://cdn.example.com",
	}
	if decoded.Social.S3 == nil || *decoded.Social.S3 != want {
		t.Fatal("Social/S3 YAML did not preserve all configuration fields")
	}
}

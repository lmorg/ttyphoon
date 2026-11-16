package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/app"
	"gopkg.in/yaml.v3"
)

func TestSocialS3YAML(t *testing.T) {
	var decoded configT
	decoder := yaml.NewDecoder(strings.NewReader(`
Social:
  S3:
    EnvFiles: [s3.vars, /secure/aws.vars]
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
		EnvFiles: []string{"s3.vars", "/secure/aws.vars"},
		Endpoint: "https://s3.example.com", Region: "us-east-1",
		AccessKeyID: "$S3_KEY", SecretAccessKey: "$S3_SECRET", SessionToken: "$S3_TOKEN",
		Bucket: "images", Prefix: "clipboard", PublicBaseURL: "https://cdn.example.com",
	}
	if decoded.Social.S3 == nil || !reflect.DeepEqual(*decoded.Social.S3, want) {
		t.Fatal("Social/S3 YAML did not preserve all configuration fields")
	}
}

func TestS3ReloadReplacesEnvironmentFilesAndRemovedFields(t *testing.T) {
	previousConfig, previousCache, previousDirs := Config, langCache, xdg.ConfigDirs
	t.Cleanup(func() { Config, langCache, xdg.ConfigDirs = previousConfig, previousCache, previousDirs })
	Config, langCache = configT{}, LanguagesT{}
	xdg.ConfigDirs = []string{t.TempDir()}
	dir := filepath.Join(xdg.ConfigDirs[0], app.DirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "test.yaml")
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("Social:\n  S3:\n    EnvFiles: [old.vars]\n    AccessKeyID: old-key\n    Bucket: old-bucket\n")
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	write("Social:\n  S3:\n    EnvFiles: [new.vars]\n    Bucket: new-bucket\n")
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if Config.Social.S3 == nil || !reflect.DeepEqual(Config.Social.S3.EnvFiles, []string{"new.vars"}) ||
		Config.Social.S3.AccessKeyID != "" || Config.Social.S3.Bucket != "new-bucket" {
		t.Fatal("reload retained old S3 environment files or literal fields")
	}
}

func TestS3ReloadFailureRestoresPreviousConfiguration(t *testing.T) {
	previousConfig, previousCache, previousDirs := Config, langCache, xdg.ConfigDirs
	t.Cleanup(func() { Config, langCache, xdg.ConfigDirs = previousConfig, previousCache, previousDirs })
	Config, langCache = configT{}, LanguagesT{}
	previous := &S3Config{Bucket: "previous"}
	Config.Social.S3 = previous
	xdg.ConfigDirs = []string{t.TempDir()}
	dir := filepath.Join(xdg.ConfigDirs[0], app.DirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "invalid.yaml"), []byte("Social: ["), 0600); err != nil {
		t.Fatal(err)
	}
	if err := LoadConfig(); err == nil || Config.Social.S3 != previous {
		t.Fatal("failed reload did not preserve the previous S3 configuration")
	}
}

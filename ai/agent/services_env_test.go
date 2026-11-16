package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lmorg/ttyphoon/config"
)

func TestServiceEnvironmentsAreScopedAndPresenceAware(t *testing.T) {
	dir := t.TempDir()
	globalFile := filepath.Join(dir, "global.vars")
	chatFile := filepath.Join(dir, "chat.vars")
	imageFile := filepath.Join(dir, "image.vars")
	for path, content := range map[string]string{
		globalFile: "SCOPED_SHARED=global\nSCOPED_GLOBAL=global-only\n",
		chatFile:   "SCOPED_SHARED=chat-file\nSCOPED_CHAT=chat-only\n",
		imageFile:  "SCOPED_SHARED=image-file\nSCOPED_IMAGE=image-only\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("SCOPED_SHARED", "host")
	t.Setenv("SCOPED_EMPTY", "host")
	t.Setenv("SCOPED_HOST", "host-only")

	previousFiles := config.Config.Ai.EnvFiles
	config.Config.Ai.EnvFiles = []string{globalFile}
	services := config.Config.Ai.Services()
	chatKey, imageKey := "test-scoped-chat", "test-scoped-image"
	previousChat, hadChat := services[chatKey]
	previousImage, hadImage := services[imageKey]
	t.Cleanup(func() {
		config.Config.Ai.EnvFiles = previousFiles
		if hadChat {
			services[chatKey] = previousChat
		} else {
			delete(services, chatKey)
		}
		if hadImage {
			services[imageKey] = previousImage
		} else {
			delete(services, imageKey)
		}
	})
	services[chatKey] = &config.AIServiceT{
		Label: chatKey, ImageGenService: imageKey, EnvFiles: []string{chatFile},
		Env: map[string]string{"SCOPED_SHARED": "chat-explicit", "SCOPED_EMPTY": ""},
	}
	services[imageKey] = &config.AIServiceT{
		Label: imageKey, EnvFiles: []string{imageFile}, Env: map[string]string{"SCOPED_SHARED": "image-explicit"},
	}
	agt := &Agent{serviceName: chatKey}
	chat, err := agt.ServiceEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	image, err := agt.ImageGenerationEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, chat, image string
	}{
		{"SCOPED_SHARED", "chat-explicit", "image-explicit"},
		{"SCOPED_GLOBAL", "global-only", "global-only"},
		{"SCOPED_EMPTY", "", "host"},
		{"SCOPED_HOST", "host-only", "host-only"},
	} {
		if got, _ := chat.Lookup(test.name); got != test.chat {
			t.Errorf("chat %s = %q, want %q", test.name, got, test.chat)
		}
		if got, _ := image.Lookup(test.name); got != test.image {
			t.Errorf("image %s = %q, want %q", test.name, got, test.image)
		}
	}
	if _, ok := image.Overlay(nil)["SCOPED_CHAT"]; ok {
		t.Fatal("chat-only value leaked into image service")
	}
	if _, ok := chat.Overlay(nil)["SCOPED_IMAGE"]; ok {
		t.Fatal("image-only value leaked into chat service")
	}
	if os.Getenv("SCOPED_SHARED") != "host" {
		t.Fatal("service resolution changed host environment")
	}

	services[chatKey].ImageGenService = ""
	image, err = agt.ImageGenerationEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := image.Lookup("SCOPED_SHARED"); got != "chat-explicit" {
		t.Fatalf("image fallback = %q, want chat service value", got)
	}
}

func TestServiceEnvironmentReportsFileErrors(t *testing.T) {
	previous := config.Config.Ai.EnvFiles
	config.Config.Ai.EnvFiles = []string{filepath.Join(t.TempDir(), "missing.vars")}
	t.Cleanup(func() { config.Config.Ai.EnvFiles = previous })
	if _, err := (&Agent{}).ServiceEnvironment(); err == nil {
		t.Fatal("missing AI environment file was not reported")
	}
	if _, err := newEinoRuntime(&Agent{serviceName: LLM_OPENAI, modelName: "test"}); err == nil {
		t.Fatal("provider initialization ignored an environment file failure")
	}
}

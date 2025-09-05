package config

import (
	"bytes"
	_ "embed"
	"io"
	"log"
	"os"
	"strings"

	"github.com/lmorg/murex/utils/lists"
	"github.com/lmorg/murex/utils/which"
	"github.com/lmorg/mxtty/utils/themes/iterm2"
	"gopkg.in/yaml.v3"
)

//go:embed defaults.yaml
var defaults []byte

func init() {
	err := ReadConfig(bytes.NewReader(defaults))
	if err != nil {
		panic(err)
	}

	files := GetFiles(".", ".yaml")
	for i := range files {
		f, err := os.Open(files[i])
		if err != nil {
			log.Println(err)
			continue
		}
		err = ReadConfig(f)
		if err != nil {
			log.Println(err)
		}
	}
}

func ReadConfig(r io.Reader) error {
	yml := yaml.NewDecoder(r)
	yml.KnownFields(true)

	err := yml.Decode(&Config)
	if err != nil {
		return err
	}

	if Config.Terminal.ColorTheme != "" {
		colorTheme := os.ExpandEnv(Config.Terminal.ColorTheme)
		home, _ := os.UserHomeDir()
		colorTheme = strings.ReplaceAll(colorTheme, "~", home)
		err = iterm2.GetTheme(colorTheme)
		if err != nil {
			panic(err)
		}
	}

	return nil
}

var Config configT

type configT struct {
	Tmux struct {
		Enabled bool `yaml:"Enabled"`
	} `yaml:"Tmux"`

	Shell struct {
		Default  []string `yaml:"Default"`
		Fallback []string `yaml:"Fallback"`
	} `yaml:"Shell"`

	Terminal struct {
		ColorTheme              string `yaml:"ColorTheme"`
		ScrollbackHistory       int    `yaml:"ScrollbackHistory"`
		ScrollbackCloseKeyPress bool   `yaml:"ScrollbackCloseKeyPress"`
		JumpScrollLineCount     int    `yaml:"JumpScrollLineCount"`
		AutoHyperlink           bool   `yaml:"AutoHyperlink"`

		Widgets struct {
			Table struct {
				ScrollMultiplierX int32 `yaml:"ScrollMultiplierX"`
				ScrollMultiplierY int32 `yaml:"ScrollMultiplierY"`
			} `yaml:"Table"`

			AutoHyperlink struct {
				IncLineNumbers bool        `yaml:"IncLineNumbers"`
				OpenAgents     OpenAgentsT `yaml:"OpenAgents"`
			} `yaml:"AutoHyperlink"`
		} `yaml:"Widgets"`
	} `yaml:"Terminal"`

	Window struct {
		Opacity                int  `yaml:"Opacity"`
		InactiveOpacity        int  `yaml:"InactiveOpacity"`
		StatusBar              bool `yaml:"StatusBar"`
		TabBarFrame            bool `yaml:"TabBarFrame"`
		TabBarActiveHighlight  bool `yaml:"TabBarActiveHighlight"`
		HoverEffectHighlight   bool `yaml:"HoverEffectHighlight"`
		TileHighlightFill      bool `yaml:"TileHighlightFill"`
		RefreshInterval        int  `yaml:"RefreshInterval"`
		UseGPU                 bool `yaml:"UseGPU"`
		BellVisualNotification bool `yaml:"BellVisualNotification"`
		BellPlayAudio          bool `yaml:"BellPlayAudio"`
	} `yaml:"Window"`

	TypeFace struct {
		FontName         string   `yaml:"FontName"`
		FontSize         int      `yaml:"FontSize"`
		Ligatures        bool     `yaml:"Ligatures"`
		LigaturePairs    []string `yaml:"LigaturePairs"`
		DropShadow       bool     `yaml:"DropShadow"`
		AdjustCellWidth  int      `yaml:"AdjustCellWidth"`
		AdjustCellHeight int      `yaml:"AdjustCellHeight"`
	} `yaml:"TypeFace"`

	Ai struct {
		AvailableModels map[string][]string `yaml:"AvailableModels"`
		DefaultModels   map[string]string   `yaml:"DefaultModels"`
		DefaultService  string              `yaml:"DefaultService"`
	} `yaml:"AI"`
}

type OpenAgentsT []struct {
	Name    string   `yaml:"Name"`
	Command []string `yaml:"Command"`
	Schemes []string `yaml:"Schemes"`
}

func (oa *OpenAgentsT) MenuItems(scheme string) (apps []string, cmds [][]string) {
	for i := range *oa {
		if !lists.Match((*oa)[i].Schemes, scheme) && !lists.Match((*oa)[i].Schemes, "*") {
			continue
		}

		if !oa.isAvail((*oa)[i].Command[0]) {
			continue
		}

		apps = append(apps, (*oa)[i].Name)

		cmd := make([]string, len((*oa)[i].Command))
		copy(cmd, (*oa)[i].Command)
		cmds = append(cmds, cmd)
	}

	return apps, cmds
}

func (oa *OpenAgentsT) isAvail(exe string) bool {
	return which.Which(exe) != ""
}

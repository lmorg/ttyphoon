package config

import (
	"bytes"
	_ "embed"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/lmorg/murex/utils/lists"
	"github.com/lmorg/murex/utils/which"
	"github.com/lmorg/ttyphoon/codes"
	"github.com/lmorg/ttyphoon/utils/file"
	"github.com/lmorg/ttyphoon/utils/themes/iterm2"
	"gopkg.in/yaml.v3"
)

//go:embed defaults.yaml
var defaults []byte

//go:embed reserved_words.yaml
var reservedWords []byte

var langCache = LanguagesT{}

func init() {
	if err := LoadConfig(); err != nil {
		panic(err)
	}
}

func LoadConfig() error {
	err := readConfigFile(bytes.NewReader(defaults))
	if err != nil {
		return err
	}

	err = readConfigFile(bytes.NewReader(reservedWords))
	if err != nil {
		return err
	}

	files := file.GetConfigFiles(".", ".yaml")
	for i := range files {
		f, err := os.Open(files[i])
		if err != nil {
			return err
		}
		err = readConfigFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func readConfigFile(r io.Reader) error {
	yml := yaml.NewDecoder(r)
	yml.KnownFields(true)

	err := yml.Decode(&Config)
	if err != nil {
		log.Println(err)
		return err
	}

	if Config.Terminal.ColorTheme != "" {
		colorTheme := os.ExpandEnv(Config.Terminal.ColorTheme)
		home, _ := os.UserHomeDir()
		colorTheme = strings.ReplaceAll(colorTheme, "~", home)
		err = iterm2.GetTheme(colorTheme)
		if err != nil {
			return err
		}
	}

	for _, custom := range Config.Terminal.Widgets.AutoHyperlink.CustomRegexp {
		custom.Rx, err = regexp.Compile(custom.Match)
		if err != nil {
			return err
		}
	}

	// Lowercase all language keys in the Languages map
	lowercasedLanguages := make(LanguagesT)
	for k, v := range Config.Notes.Languages {
		lowercasedLanguages[strings.ToLower(k)] = v
	}
	Config.Notes.Languages = lowercasedLanguages

	// merge reserved words
	for k, v := range langCache {
		if Config.Notes.Languages[k] == nil {
			Config.Notes.Languages[k] = &LanguageOptionsT{}
		}
		for _, word := range v.ReservedWords {
			Config.Notes.Languages[k].ReservedWords = append(Config.Notes.Languages[k].ReservedWords, word)
		}
	}
	for k, v := range Config.Notes.Languages {
		if langCache[k] == nil {
			langCache[k] = &LanguageOptionsT{}
		}
		langCache[k].ReservedWords = v.ReservedWords
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

	Hotkeys struct {
		PrefixTtl int              `yaml:"PrefixTTL"`
		RepeatTtl int              `yaml:"RepeatTTL"`
		Functions HotkeyFunctionsT `yaml:"Functions"`
	} `yaml:"Hotkeys"`

	Terminal struct {
		ColorTheme              string `yaml:"ColorTheme"`
		ScrollbackHistory       int    `yaml:"ScrollbackHistory"`
		ScrollbackCloseKeyPress bool   `yaml:"ScrollbackCloseKeyPress"`
		WriteMarkdownHistory    bool   `yaml:"WriteMarkdownHistory"`
		JumpScrollLineCount     int    `yaml:"JumpScrollLineCount"`
		AutoHyperlink           bool   `yaml:"AutoHyperlink"`

		Widgets struct {
			Table struct {
				ScrollMultiplierX int32 `yaml:"ScrollMultiplierX"`
				ScrollMultiplierY int32 `yaml:"ScrollMultiplierY"`
			} `yaml:"Table"`

			AutoHyperlink struct {
				IncLineNumbers bool                          `yaml:"IncLineNumbers"`
				OpenAgents     OpenAgentsT                   `yaml:"OpenAgents"`
				CustomRegexp   []*AutoHyperlinkCustomRegexpT `yaml:"CustomRegexp"`
			} `yaml:"AutoHyperlink"`
		} `yaml:"Widgets"`
	} `yaml:"Terminal"`

	Window struct {
		StatusBar              bool `yaml:"StatusBar"`
		HoverEffectHighlight   bool `yaml:"HoverEffectHighlight"`
		RefreshInterval        int  `yaml:"RefreshInterval"`
		BellVisualNotification bool `yaml:"BellVisualNotification"`
		BellPlayAudio          bool `yaml:"BellPlayAudio"`
		AlwaysOnTop            bool `yaml:"AlwaysOnTop"`
	} `yaml:"Window"`

	Notes struct {
		MaxFileSize       int64 `yaml:"MaxFileSizeMB"`
		StructViewMaxSize int64 `yaml:"StructViewMaxSizeKB"`
		MaxRecentFiles    int   `yaml:"MaxRecentFiles"`
		MaxLogLines       int   `yaml:"MaxLogLines"`

		Languages LanguagesT `yaml:"Languages"`

		SpellCheck struct {
			// TyposLsp configures the language-agnostic `typos-lsp` server used
			// for spellchecking project files and Jupyter code blocks when LSP
			// mode is enabled. When the command is empty or the server cannot
			// start, Notes falls back to the aspell spellchecker.
			TyposLsp LspT `yaml:"TyposLsp"`
		} `yaml:"SpellCheck"`
	} `yaml:"Notes"`

	TypeFace struct {
		FontName         string `yaml:"FontName"`
		FontSize         int    `yaml:"FontSize"`
		Ligatures        bool   `yaml:"Ligatures"`
		AdjustCellWidth  int    `yaml:"AdjustCellWidth"`
		AdjustCellHeight int    `yaml:"AdjustCellHeight"`
	} `yaml:"TypeFace"`

	Ai struct {
		MaxIterations   int                 `yaml:"MaxIterations"`
		AvailableModels map[string][]string `yaml:"AvailableModels"`
		DefaultModels   map[string]string   `yaml:"DefaultModels"`
		DefaultService  string              `yaml:"DefaultService"`
	} `yaml:"AI"`
}

type LanguagesT map[string]*LanguageOptionsT

type LanguageOptionsT struct {
	TabSpaceIndent int      `yaml:"TabSpaceIndent"`
	LSP            LspT     `yaml:"LSP"`
	ReservedWords  []string `yaml:"ReservedWords"`
}

type LspT struct {
	Command     []string       `yaml:"command"`
	InitOptions map[string]any `yaml:"initializationOptions"`
}

// GetTabIndent returns the number of spaces to use for indentation for a given language.
// If the language is not found, it falls back to the "default" key.
func (lt LanguagesT) GetTabIndent(language string) int {
	lang := strings.ToLower(language)
	if lang == "" {
		lang = "default"
	}

	if settings, ok := lt[lang]; ok {
		if settings.TabSpaceIndent > 0 {
			return settings.TabSpaceIndent
		}
	}

	// Fallback to default
	if settings, ok := lt["default"]; ok {
		if settings.TabSpaceIndent > 0 {
			return settings.TabSpaceIndent
		}
	}

	// Final fallback if nothing is configured
	return 4
}

type HotkeyFunctionsT map[string]string

func (hk HotkeyFunctionsT) Scan() []*hotkeyFunctionT {
	var hotkeysFunctions []*hotkeyFunctionT
	for hotkeys, function := range hk {
		keys := strings.SplitN(hotkeys, "::", 2)
		switch len(keys) {
		case 1:
			hotkeysFunctions = append(hotkeysFunctions, &hotkeyFunctionT{
				Function: function,
				Hotkey:   codes.KeyName(hotkeys),
			})

		case 2:
			hotkeysFunctions = append(hotkeysFunctions, &hotkeyFunctionT{
				Function: function,
				Prefix:   codes.KeyName(keys[0]),
				Hotkey:   codes.KeyName(keys[1]),
			})

		default:
			panic(hotkeys)
		}
	}
	return hotkeysFunctions
}

type hotkeyFunctionT struct {
	Function string
	Prefix   codes.KeyName
	Hotkey   codes.KeyName
}

type OpenAgentsT []struct {
	Name    string   `yaml:"Name"`
	Command []string `yaml:"Command"`
	Schemes []string `yaml:"Schemes"`
}

type AutoHyperlinkCustomRegexpT struct {
	Match string         `yaml:"Match"`
	Link  string         `yaml:"Link"`
	Rx    *regexp.Regexp `yaml:"-"`
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

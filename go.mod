module github.com/lmorg/mxtty

go 1.24.1

require (
	github.com/creack/pty v1.1.24
	github.com/flopp/go-findfont v0.1.0
	github.com/forPelevin/gomoji v1.3.0
	github.com/go-text/render v0.2.0
	github.com/go-text/typesetting v0.3.0
	github.com/lmorg/murex v0.0.0-20250115225944-b4c429617fd4
	github.com/lmorg/readline/v4 v4.1.0
	github.com/mattn/go-runewidth v0.0.16
	github.com/mattn/go-sixel v0.0.5
	github.com/mattn/go-sqlite3 v1.14.24
	github.com/pkoukk/tiktoken-go v0.1.7
	github.com/tmc/langchaingo v0.1.13
	github.com/veandco/go-sdl2 v0.5.0-alpha.7
	golang.design/x/clipboard v0.7.0
	golang.design/x/hotkey v0.4.1
	golang.org/x/image v0.24.0
	golang.org/x/sys v0.32.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/dlclark/regexp2 v1.10.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/soniakeys/quant v1.0.0 // indirect
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c // indirect
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef // indirect
	golang.org/x/exp/shiny v0.0.0-20250506013437-ce4c2cf36ca6 // indirect
	golang.org/x/mobile v0.0.0-20231127183840-76ac6878050a // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
)

replace github.com/lmorg/readline/v4 => ../readline

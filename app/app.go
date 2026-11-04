package app

// DirName is the name in which config files et al are stored
const DirName = "ttyphoon"

var (
	name      = "Ttyphoon"
	tagLine   = "The Next Generation Terminal Emulator"
	version   = "1.0.1000"
	branch    = "unknown"
	buildDate = "unknown"
	copyright = "2023-2026 Laurence Morgan"
	license   = "GPL v2"
)

func Name() string      { return name }
func TagLine() string   { return tagLine }
func Version() string   { return version }
func Branch() string    { return branch }
func BuildDate() string { return buildDate }
func Copyright() string { return copyright }
func License() string   { return license }

const ProjectSourcePath = "github.com/lmorg/ttyphoon/"

const WebSite = "https://github.com/lmorg/ttyphoon"

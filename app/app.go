package app

// DirName is the name in which config files et al are stored
const DirName = "ttyphoon"

var (
	name      = "undef"
	tagLine   = "undef"
	version   = "undef"
	branch    = "undef"
	buildDate = "undef"
	copyright = "undef"
	license   = "undef"
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

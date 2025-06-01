package types

type Pty interface {
	ExecuteShell(exit func())
	Read() (rune, error)
	Write([]byte) error
	Resize(*XY) error
	BufSize() int
	Close()
}

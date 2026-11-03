package workspace

import "os"

func (t *TermT) PwdTmux() string {
	return PwdTmux(t.Pwd)
}

func PwdTmux(pwd string) string {
	return QuoteString(os.ExpandEnv(pwd))
}

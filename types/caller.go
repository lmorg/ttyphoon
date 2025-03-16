package types

type CallerT int

const (
	CALLER_updateWinInfo CallerT = 1 + iota
	CALLER__respWindowAdd
)

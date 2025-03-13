package getshell

func GetShell() string {
	//» dscl . -read /Users/$USER UserShell
	//UserShell: "/bin/zsh"
	return "/bin/zsh"
}

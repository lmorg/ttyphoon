package getshell

import (
	"bytes"
	"errors"
	"os/exec"
	"os/user"

	"gopkg.in/yaml.v3"
)

type dsclT struct {
	UserShell string `yaml:"UserShell"`
}

func dscl() (string, error) {
	//» dscl . -read /Users/$USER UserShell
	//UserShell: "/bin/zsh"

	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	cmd := exec.Command("dscl", ".", "-read", usr.HomeDir, "UserShell")

	buf := new(bytes.Buffer)
	cmd.Stdout = buf

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	err = cmd.Wait()
	if err != nil {
		return "", err
	}

	shell := new(dsclT)
	err = yaml.Unmarshal(buf.Bytes(), &shell)
	if err != nil {
		return "", err
	}

	if shell.UserShell == "" {
		return "", errors.New("empty shell string but no error raised")
	}

	return shell.UserShell, nil
}

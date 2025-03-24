package getshell

import (
	"bytes"
	"errors"
	"os/exec"
	"os/user"
	"strings"
)

func getent() (string, error) {
	//» getent passwd alice
	//alice:x:1001:1001:Alice:/home/alice:/bin/zsh

	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	cmd := exec.Command("getent", "passwd", usr.Username)

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

	passwd := strings.Split(buf.String(), ":")
	if len(passwd) != 7 {
		return "", errors.New("too many or too few fields returned from getent")
	}

	if passwd[6] == "" {
		return "", errors.New("empty getent field but no error raised")
	}

	return passwd[6], nil
}

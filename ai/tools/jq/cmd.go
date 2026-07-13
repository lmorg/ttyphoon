package jq

import (
	"bytes"
	"os/exec"
	"strings"
)

func execJq(json string, query string) (string, error) {
	cmd := exec.Command(`jq`, query)

	cmd.Stdin = strings.NewReader(json)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil && stderr.Len() == 0 {
		return "", err
	}
	sOut := strings.TrimSpace(stdout.String())

	sErr := strings.TrimSpace(stderr.String())
	if sErr != "" {
		s := strings.Split(sErr, "\n")
		sErr = "error: " + strings.Join(s, "\nerror: ") + "\n"
	}

	return sOut + sErr, nil
}

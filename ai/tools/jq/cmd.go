package jq

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func execJq(json string, query string) (string, error) {
	cmd := exec.Command(`jq`, query)

	cmd.Stdin = strings.NewReader(json)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("cannot create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("cannot create stderr pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	_ = cmd.Wait()

	bOut, err := io.ReadAll(stdout)
	if err != nil {
		return "", fmt.Errorf("cannot read from stdout pipe: %w", err)
	}

	bErr, err := io.ReadAll(stderr)
	if err != nil {
		return "", fmt.Errorf("cannot read from stderr pipe: %w", err)
	}

	sOut := strings.TrimSpace(string(bOut))

	sErr := strings.TrimSpace(string(bErr))
	if sErr != "" {
		s := strings.Split(sErr, "\n")
		sErr = "error: " + strings.Join(s, "\nerror: ") + "\n"
	}

	return sOut + sErr, nil
}

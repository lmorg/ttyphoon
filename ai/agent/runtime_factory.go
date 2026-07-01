package agent

import "context"

type runtimeInitError struct {
	err error
}

func (r *runtimeInitError) RunLLMWithStream(_ context.Context, _ string, _ func(string)) (string, error) {
	return "", r.err
}

func (r *runtimeInitError) Reset() {}

func newPreferredRuntime(agent *Agent) agentRuntime {
	rt, err := newEinoRuntime(agent)
	if err != nil {
		return &runtimeInitError{err: err}
	}

	return rt
}

package agent

type client interface {
	Close() error
}

const _MAX_ITERATIONS = 30

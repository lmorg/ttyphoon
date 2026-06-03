package types

type NotificationType int

const (
	NOTIFY_DEBUG = iota
	NOTIFY_INFO
	NOTIFY_WARN
	NOTIFY_ERROR
	NOTIFY_QUESTION
)

type Notification interface {
	SetMessage(string)
	UpdateCanceller(func())
	Close()
}

func (n NotificationType) String() string {
	switch n {
	case NOTIFY_DEBUG:
		return "debug"
	case NOTIFY_INFO:
		return "info"
	case NOTIFY_WARN:
		return "warn"
	case NOTIFY_ERROR:
		return "error"
	case NOTIFY_QUESTION:
		return "question"
	default:
		return "unknown"
	}
}

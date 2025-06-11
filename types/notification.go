package types

type NotificationType int

const (
	NOTIFY_DEBUG = iota
	NOTIFY_INFO
	NOTIFY_WARN
	NOTIFY_ERROR
	NOTIFY_SCROLL
	NOTIFY_QUESTION
)

type Notification interface {
	SetMessage(string)
	UpdateCanceller(func())
	Close()
}

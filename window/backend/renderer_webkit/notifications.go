package rendererwebkit

import (
	"context"
	"log"
	"slices"
	"sync"
	"time"

	"github.com/lmorg/ttyphoon/types"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type notifyT struct {
	timed  []*notificationT
	sticky []*notificationT
	mutex  sync.Mutex
}

type notificationPayload struct {
	ID      int64                  `json:"id"`
	Type    types.NotificationType `json:"type"`
	Message string                 `json:"message"`
	Sticky  bool                   `json:"sticky"`
}

type notificationT struct {
	id     int64
	typ    types.NotificationType
	msg    string
	sticky bool
	ctx    context.Context
	cancel func()
	closed bool
	wr     *webkitRender
}

func (nt *notificationT) SetMessage(message string) {
	nt.msg = message
	nt.emit()
}

func (nt *notificationT) UpdateCanceller(cancel func()) {
	nt.cancel = cancel
}

func (nt *notificationT) Close() {
	nt.closed = true
	if nt.cancel != nil {
		nt.cancel()
	}
	if nt.wr != nil && nt.wr.wapp != nil {
		runtime.EventsEmit(nt.wr.wapp, "terminalNotificationClose", nt.id)
	}
}

func (nt *notificationT) emit() {
	if nt.wr == nil || nt.wr.wapp == nil {
		return
	}
	runtime.EventsEmit(nt.wr.wapp, "terminalNotification", notificationPayload{
		ID:      nt.id,
		Type:    nt.typ,
		Message: nt.msg,
		Sticky:  nt.sticky,
	})
}

func (wr *webkitRender) DisplayNotification(notificationType types.NotificationType, message string) {
	nt := &notificationT{
		id:  time.Now().UnixMilli(),
		typ: notificationType,
		msg: message,
		wr:  wr,
	}
	wr.notifications.addTimed(nt)
}

func (wr *webkitRender) DisplaySticky(notificationType types.NotificationType, message string, cancel func()) types.Notification {
	nt := &notificationT{
		id:     time.Now().UnixMilli(),
		typ:    notificationType,
		msg:    message,
		sticky: true,
		cancel: cancel,
		wr:     wr,
	}
	wr.notifications.addSticky(nt)
	return nt
}

func (n *notifyT) addTimed(nt *notificationT) {
	const d = 5 * time.Second
	nt.ctx, nt.cancel = context.WithTimeout(context.Background(), d)

	n.mutex.Lock()
	n.timed = append(n.timed, nt)
	n.mutex.Unlock()

	nt.emit()
	log.Printf("NOTIFICATION: %s", nt.msg)

	go func() {
		<-nt.ctx.Done()
		nt.closed = true
		n.delete(nt)
		if nt.wr != nil && nt.wr.wapp != nil {
			runtime.EventsEmit(nt.wr.wapp, "terminalNotificationClose", nt.id)
		}
	}()
}

func (n *notifyT) addSticky(nt *notificationT) {
	n.mutex.Lock()
	n.sticky = append(n.sticky, nt)
	n.mutex.Unlock()

	nt.emit()
	log.Printf("NOTIFICATION: %s", nt.msg)
}

func (n *notifyT) delete(nt *notificationT) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	var notifications *[]*notificationT
	if nt.sticky {
		notifications = &n.sticky
	} else {
		notifications = &n.timed
	}

	for i := range *notifications {
		if (*notifications)[i].id == nt.id {
			*notifications = slices.Delete(*notifications, i, i+1)
			return
		}
	}
}

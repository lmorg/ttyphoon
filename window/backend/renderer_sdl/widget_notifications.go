package rendersdl

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/lmorg/mxtty/assets"
	"github.com/lmorg/mxtty/types"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

var _notifyColourLight = map[int]*types.Colour{
	types.NOTIFY_DEBUG:  {0x31, 0x6d, 0xb0, 223},
	types.NOTIFY_INFO:   {0x99, 0xc0, 0xd3, 223},
	types.NOTIFY_WARN:   {0xf2, 0xb7, 0x1f, 223},
	types.NOTIFY_ERROR:  {0xde, 0x33, 0x3b, 223},
	types.NOTIFY_SCROLL: {0x31, 0x6d, 0xb0, 223},
}

var _notifyColourDark = map[int]*types.Colour{
	types.NOTIFY_DEBUG:  {0x1c, 0x3e, 0x64, 223},
	types.NOTIFY_INFO:   {0x31, 0x6d, 0xb0, 223},
	types.NOTIFY_WARN:   {0x74, 0x58, 0x10, 223},
	types.NOTIFY_ERROR:  {0x66, 0x16, 0x1a, 223},
	types.NOTIFY_SCROLL: {0x1c, 0x3e, 0x64, 223},
}

var notifyColour, notifyBorderColour map[int]*types.Colour

func (sr *sdlRender) preloadNotificationGlyphs() {
	var err error
	sr.notifyIcon = map[int]types.Image{
		types.NOTIFY_DEBUG:    nil,
		types.NOTIFY_INFO:     nil,
		types.NOTIFY_WARN:     nil,
		types.NOTIFY_ERROR:    nil,
		types.NOTIFY_QUESTION: nil,
	}
	sr.notifyIconSize = &types.XY{
		X: sr.glyphSize.Y + (_WIDGET_INNER_MARGIN * 4),
		Y: sr.glyphSize.Y + (_WIDGET_INNER_MARGIN * 4),
	}

	sr.notifyIcon[types.NOTIFY_DEBUG], err = sr.loadImage(assets.Get(assets.ICON_DEBUG), sr.notifyIconSize)
	if err != nil {
		panic(err)
	}

	sr.notifyIcon[types.NOTIFY_INFO], err = sr.loadImage(assets.Get(assets.ICON_INFO), sr.notifyIconSize)
	if err != nil {
		panic(err)
	}

	sr.notifyIcon[types.NOTIFY_WARN], err = sr.loadImage(assets.Get(assets.ICON_WARN), sr.notifyIconSize)
	if err != nil {
		panic(err)
	}

	sr.notifyIcon[types.NOTIFY_ERROR], err = sr.loadImage(assets.Get(assets.ICON_ERROR), sr.notifyIconSize)
	if err != nil {
		panic(err)
	}

	sr.notifyIcon[types.NOTIFY_SCROLL], err = sr.loadImage(assets.Get(assets.ICON_DOWN), sr.notifyIconSize)
	if err != nil {
		panic(err)
	}

	sr.notifyIcon[types.NOTIFY_QUESTION], err = sr.loadImage(assets.Get(assets.ICON_QUESTION), sr.notifyIconSize)
	if err != nil {
		panic(err)
	}
}

type notifyT struct {
	timed  []*notificationT
	sticky []*notificationT
	mutex  sync.Mutex
}

type notificationT struct {
	Type    types.NotificationType
	Message string
	wait    <-chan time.Time
	end     time.Time
	close   func()
	id      int64
	//paneId  string
}

func (notification *notificationT) SetMessage(message string) {
	notification.Message = message
}

func (notification *notificationT) Close() {
	if notification.close != nil {
		notification.close()
	}
}

func (n *notifyT) _wait() {
	for {
		if len(n.timed) == 0 {
			return
		}

		<-n.timed[0].wait
		n.remove()
	}
}

func (n *notifyT) addTimed(notification *notificationT) {
	d := 5 * time.Second
	notification.end = time.Now().Add(d)
	notification.wait = time.After(d)

	n.mutex.Lock()
	n.timed = append(n.timed, notification)

	if len(n.timed) > 0 {
		go n._wait()
	}
	n.mutex.Unlock()

	log.Printf("NOTIFICATION: %s", notification.Message)
}

func (n *notifyT) addSticky(notification *notificationT) {
	notification.id = time.Now().UnixMilli()
	notification.close = func() {
		n.mutex.Lock()
		var i int
		for i := range n.sticky {
			if n.sticky[i].id == notification.id {
				goto matched
			}
		}
		return
	matched:
		n.sticky = append(n.sticky[:i], n.sticky[i+1:]...)
		n.mutex.Unlock()
	}

	n.mutex.Lock()
	n.sticky = append(n.sticky, notification)
	n.mutex.Unlock()

	log.Printf("NOTIFICATION: %s", notification.Message)
}

func (n *notifyT) remove() {
	n.mutex.Lock()
	n.timed = n.timed[1:]
	n.mutex.Unlock()
}

func (n *notifyT) get() []*notificationT {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if len(n.sticky) == 0 && len(n.timed) == 0 {
		return nil
	}

	notifications := make([]*notificationT, len(n.timed)+len(n.sticky))
	copy(notifications, n.sticky)
	copy(notifications[len(n.sticky):], n.timed)

	return notifications
}

func (sr *sdlRender) DisplayNotification(notificationType types.NotificationType, message string) {
	notification := &notificationT{
		Type:    notificationType,
		Message: message,
		//paneId:  sr.tmux.ActivePane().Id,
	}
	sr.notifications.addTimed(notification)
}

func (sr *sdlRender) DisplaySticky(notificationType types.NotificationType, message string) types.Notification {
	notification := &notificationT{
		Type:    notificationType,
		Message: message,
		//paneId:  sr.tmux.ActivePane().Id,
	}
	sr.notifications.addSticky(notification)

	return notification
}

func (sr *sdlRender) renderNotification(windowRect *sdl.Rect) {
	notifications := sr.notifications.get()
	if notifications == nil {
		return
	}

	surface, err := sdl.CreateRGBSurfaceWithFormat(0, windowRect.W, windowRect.H, 32, uint32(sdl.PIXELFORMAT_RGBA32))
	if err != nil {
		panic(err) // TODO: don't panic!
	}
	defer surface.Free()

	sr.font.SetStyle(ttf.STYLE_BOLD)

	var offset int32
	for _, notification := range notifications {
		//if notification.paneId != "" && notification.paneId != sr.tmux.ActivePane().Id {
		//	continue
		//}

		// generate text
		s := strconv.Itoa(int(time.Until(notification.end)/time.Second) + 1)
		countdown, err := sr.font.RenderUTF8Blended(s, sdl.Color{R: 0, G: 0, B: 0, A: 192})
		if err != nil {
			panic(err) // TODO: don't panic!
		}
		defer countdown.Free()

		text, err := sr.font.RenderUTF8BlendedWrapped(notification.Message, sdl.Color{R: 255, G: 255, B: 255, A: 255}, int(surface.W-sr.notifyIconSize.X-countdown.W-sr.glyphSize.X))
		if err != nil {
			panic(err) // TODO: don't panic!
		}
		defer text.Free()

		textShadow, err := sr.font.RenderUTF8BlendedWrapped(notification.Message, sdl.Color{R: 0, G: 0, B: 0, A: 150}, int(surface.W-sr.notifyIconSize.X-countdown.W-sr.glyphSize.X))
		if err != nil {
			panic(err) // TODO: don't panic!
		}
		defer textShadow.Free()

		// draw border
		bc := notifyBorderColour[int(notification.Type)]
		sr.renderer.SetDrawColor(bc.Red, bc.Green, bc.Blue, bc.Alpha)
		rect := sdl.Rect{
			X: _WIDGET_INNER_MARGIN - 1,
			Y: _WIDGET_INNER_MARGIN + offset - 1,
			W: windowRect.W - _WIDGET_OUTER_MARGIN + 2,
			H: text.H + _WIDGET_OUTER_MARGIN + 2,
		}
		sr.renderer.DrawRect(&rect)
		rect = sdl.Rect{
			X: _WIDGET_INNER_MARGIN,
			Y: _WIDGET_INNER_MARGIN + offset,
			W: windowRect.W - _WIDGET_OUTER_MARGIN,
			H: text.H + _WIDGET_OUTER_MARGIN,
		}
		sr.renderer.DrawRect(&rect)

		// fill background
		c := notifyColour[int(notification.Type)]
		sr.renderer.SetDrawColor(c.Red, c.Green, c.Blue, c.Alpha)
		rect = sdl.Rect{
			X: _WIDGET_INNER_MARGIN + 1,
			Y: _WIDGET_INNER_MARGIN + 1 + offset,
			W: surface.W - _WIDGET_OUTER_MARGIN - 2,
			H: text.H + _WIDGET_OUTER_MARGIN - 2,
		}
		sr.renderer.FillRect(&rect)

		// render countdown
		if notification.close == nil {
			rect = sdl.Rect{
				X: windowRect.W - _WIDGET_OUTER_MARGIN - countdown.W,
				Y: _WIDGET_OUTER_MARGIN + offset,
				W: countdown.W,
				H: countdown.H,
			}
			err = countdown.Blit(nil, surface, &rect)
			if err != nil {
				panic(err) // TODO: don't panic!
			}
			sr._renderNotificationSurface(surface, &rect)
		}

		// render shadow
		rect = sdl.Rect{
			X: _WIDGET_OUTER_MARGIN + sr.notifyIconSize.X + sr.glyphSize.X + 2,
			Y: _WIDGET_OUTER_MARGIN + offset + 2,
			W: surface.W - sr.notifyIconSize.X - countdown.W,
			H: text.H + _WIDGET_OUTER_MARGIN - 2,
		}
		_ = textShadow.Blit(nil, surface, &rect)
		sr._renderNotificationSurface(surface, &rect)

		// render text
		rect = sdl.Rect{
			X: _WIDGET_OUTER_MARGIN + sr.notifyIconSize.X + sr.glyphSize.X,
			Y: _WIDGET_OUTER_MARGIN + offset,
			W: surface.W - sr.notifyIconSize.X - countdown.W,
			H: text.H + _WIDGET_OUTER_MARGIN - 2,
		}
		err = text.Blit(nil, surface, &rect)
		if err != nil {
			panic(err) // TODO: don't panic!
		}
		sr._renderNotificationSurface(surface, &rect)

		if surface, ok := sr.notifyIcon[int(notification.Type)].Asset().(*sdl.Surface); ok {
			srcRect := &sdl.Rect{
				X: 0,
				Y: 0,
				W: surface.W,
				H: surface.H,
			}

			dstRect := &sdl.Rect{
				X: _WIDGET_INNER_MARGIN * 2,
				Y: offset + ((text.H + _WIDGET_OUTER_MARGIN + _WIDGET_OUTER_MARGIN + 2) / 2) - (sr.notifyIconSize.Y / 2),
				W: sr.notifyIconSize.X,
				H: sr.notifyIconSize.X,
			}

			texture, err := sr.renderer.CreateTextureFromSurface(surface)
			if err != nil {
				panic(err) //TODO: don't panic!
			}
			defer texture.Destroy()

			err = sr.renderer.Copy(texture, srcRect, dstRect)
			if err != nil {
				panic(err) //TODO: don't panic!
			}
		}

		offset += text.H + (_WIDGET_INNER_MARGIN * 3)
	}
}

func (sr *sdlRender) _renderNotificationSurface(surface *sdl.Surface, rect *sdl.Rect) {
	texture, err := sr.renderer.CreateTextureFromSurface(surface)
	if err != nil {
		panic(err) //TODO: don't panic!
	}
	defer texture.Destroy()

	err = sr.renderer.Copy(texture, rect, rect)
	if err != nil {
		panic(err) //TODO: don't panic!
	}
}

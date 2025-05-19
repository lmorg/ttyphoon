package rendersdl

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/lmorg/mxtty/assets"
	"github.com/lmorg/mxtty/types"
	"github.com/veandco/go-sdl2/sdl"
)

var _notifyColourSchemeLight = map[types.NotificationType]*types.Colour{
	types.NOTIFY_DEBUG:    {0x31, 0x6d, 0xb0, 255},
	types.NOTIFY_INFO:     {0x99, 0xc0, 0xd3, 255},
	types.NOTIFY_WARN:     {0xf2, 0xb7, 0x1f, 255},
	types.NOTIFY_ERROR:    {0xde, 0x33, 0x3b, 255},
	types.NOTIFY_SCROLL:   {0x31, 0x6d, 0xb0, 255},
	types.NOTIFY_QUESTION: {0x74, 0x95, 0x3c, 255},
}

var _notifyColourSchemeDark = map[types.NotificationType]*types.Colour{
	types.NOTIFY_DEBUG:    {0x31, 0x6d, 0xb0, 92},
	types.NOTIFY_INFO:     {0x99, 0xc0, 0xd3, 92},
	types.NOTIFY_WARN:     {0xf2, 0xb7, 0x1f, 92},
	types.NOTIFY_ERROR:    {0xde, 0x33, 0x3b, 92},
	types.NOTIFY_SCROLL:   {0x31, 0x6d, 0xb0, 92},
	types.NOTIFY_QUESTION: {0x74, 0x95, 0x3c, 92},
}

var _notifyColourLight = map[types.NotificationType]*types.Colour{
	types.NOTIFY_DEBUG:    {0x00, 0x00, 0x00, 255},
	types.NOTIFY_INFO:     {0x00, 0x00, 0x00, 255},
	types.NOTIFY_WARN:     {0x00, 0x00, 0x00, 255},
	types.NOTIFY_ERROR:    {0x00, 0x00, 0x00, 255},
	types.NOTIFY_SCROLL:   {0x00, 0x00, 0x00, 255},
	types.NOTIFY_QUESTION: {0x00, 0x00, 0x00, 255},
}

var _notifyColourDark = map[types.NotificationType]*types.Colour{
	types.NOTIFY_DEBUG:    {0x31, 0x6d, 0xb0, 255},
	types.NOTIFY_INFO:     {0x99, 0xc0, 0xd3, 255},
	types.NOTIFY_WARN:     {0xf2, 0xb7, 0x1f, 255},
	types.NOTIFY_ERROR:    {0xde, 0x33, 0x3b, 255},
	types.NOTIFY_SCROLL:   {0x31, 0x6d, 0xb0, 255},
	types.NOTIFY_QUESTION: {0x74, 0x95, 0x3c, 255},
}

var _notifyColourSgrLight = map[types.NotificationType]*types.Sgr{
	types.NOTIFY_DEBUG: {
		Fg:      &types.Colour{0x00, 0x00, 0x00, 255},
		Bg:      &types.Colour{0x31, 0x6d, 0xb0, 192},
		Bitwise: types.SGR_BOLD,
	},
	types.NOTIFY_INFO: {
		Fg:      &types.Colour{0x00, 0x00, 0x00, 255},
		Bg:      &types.Colour{0x99, 0xc0, 0xd3, 192},
		Bitwise: types.SGR_BOLD,
	},
	types.NOTIFY_WARN: {
		Fg:      &types.Colour{0x00, 0x00, 0x00, 255},
		Bg:      &types.Colour{0xf2, 0xb7, 0x1f, 192},
		Bitwise: types.SGR_BOLD,
	},
	types.NOTIFY_ERROR: {
		Fg:      &types.Colour{0x00, 0x00, 0x00, 255},
		Bg:      &types.Colour{0xde, 0x33, 0x3b, 192},
		Bitwise: types.SGR_BOLD,
	},
	types.NOTIFY_SCROLL: {
		Fg:      &types.Colour{0x00, 0x00, 0x00, 255},
		Bg:      &types.Colour{0x31, 0x6d, 0xb0, 192},
		Bitwise: types.SGR_BOLD,
	},
	types.NOTIFY_QUESTION: {
		Fg:      &types.Colour{0x00, 0x00, 0x00, 255},
		Bg:      &types.Colour{0x74, 0x95, 0x3c, 192},
		Bitwise: types.SGR_BOLD,
	},
}

var _notifyColourSgrDark = map[types.NotificationType]*types.Sgr{
	types.NOTIFY_DEBUG: {
		Fg:      types.SGR_DEFAULT.Fg,
		Bg:      types.SGR_DEFAULT.Bg,
		Bitwise: types.SGR_BOLD,
	},
	types.NOTIFY_INFO: {
		Fg:      types.SGR_DEFAULT.Fg,
		Bg:      types.SGR_DEFAULT.Bg,
		Bitwise: types.SGR_BOLD,
	},
	types.NOTIFY_WARN: {
		Fg:      types.SGR_DEFAULT.Fg,
		Bg:      types.SGR_DEFAULT.Bg,
		Bitwise: types.SGR_BOLD,
	},
	types.NOTIFY_ERROR: {
		Fg:      types.SGR_DEFAULT.Fg,
		Bg:      types.SGR_DEFAULT.Bg,
		Bitwise: types.SGR_BOLD,
	},
	types.NOTIFY_SCROLL: {
		Fg:      types.SGR_DEFAULT.Fg,
		Bg:      types.SGR_DEFAULT.Bg,
		Bitwise: types.SGR_BOLD,
	},
	types.NOTIFY_QUESTION: {
		Fg:      types.SGR_DEFAULT.Fg,
		Bg:      types.SGR_DEFAULT.Bg,
		Bitwise: types.SGR_BOLD,
	},
}

//const _NOTIFY_ALPHA_BLEND = 215

var _notifyCountdownSgr = map[types.NotificationType]*types.Sgr{
	types.NOTIFY_DEBUG: {
		Fg:      types.SGR_DEFAULT.Fg,
		Bg:      types.SGR_DEFAULT.Bg,
		Bitwise: types.SGR_FAINT,
	},
	types.NOTIFY_INFO: {
		Fg:      types.SGR_DEFAULT.Fg,
		Bg:      types.SGR_DEFAULT.Bg,
		Bitwise: types.SGR_FAINT,
	},
	types.NOTIFY_WARN: {
		Fg:      types.SGR_DEFAULT.Fg,
		Bg:      types.SGR_DEFAULT.Bg,
		Bitwise: types.SGR_FAINT,
	},
	types.NOTIFY_ERROR: {
		Fg:      types.SGR_DEFAULT.Fg,
		Bg:      types.SGR_DEFAULT.Bg,
		Bitwise: types.SGR_FAINT,
	},
	types.NOTIFY_SCROLL: {
		Fg:      types.SGR_DEFAULT.Fg,
		Bg:      types.SGR_DEFAULT.Bg,
		Bitwise: types.SGR_FAINT,
	},
}

var (
	notifyColour       map[types.NotificationType]*types.Colour
	notifyBorderColour map[types.NotificationType]*types.Colour
	notifyColourSgr    map[types.NotificationType]*types.Sgr
)

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

	//sr.font.SetStyle(ttf.STYLE_BOLD)

	var offset int32
	for _, notification := range notifications {
		//if notification.paneId != "" && notification.paneId != sr.tmux.ActivePane().Id {
		//	continue
		//}

		textHeight := sr.glyphSize.Y
		countdownW := sr.glyphSize.X

		// draw border
		bc := notifyBorderColour[notification.Type]
		sr.renderer.SetDrawColor(bc.Red, bc.Green, bc.Blue, bc.Alpha)
		rect := sdl.Rect{
			X: _WIDGET_INNER_MARGIN - 1,
			Y: _WIDGET_INNER_MARGIN + offset - 1,
			W: windowRect.W - _WIDGET_OUTER_MARGIN + 2,
			H: textHeight + _WIDGET_OUTER_MARGIN + 2,
		}
		sr.renderer.DrawRect(&rect)
		rect = sdl.Rect{
			X: _WIDGET_INNER_MARGIN,
			Y: _WIDGET_INNER_MARGIN + offset,
			W: windowRect.W - _WIDGET_OUTER_MARGIN,
			H: textHeight + _WIDGET_OUTER_MARGIN,
		}
		sr.renderer.DrawRect(&rect)

		// fill background
		rect = sdl.Rect{
			X: _WIDGET_INNER_MARGIN + 1,
			Y: _WIDGET_INNER_MARGIN + 1 + offset,
			W: surface.W - _WIDGET_OUTER_MARGIN - 2,
			H: textHeight + _WIDGET_OUTER_MARGIN - 2,
		}
		c := types.SGR_COLOR_BACKGROUND
		sr.renderer.SetDrawColor(c.Red, c.Green, c.Blue, 255)
		sr.renderer.FillRect(&rect)
		c = notifyColour[notification.Type]
		sr.renderer.SetDrawColor(c.Red, c.Green, c.Blue, c.Alpha)
		sr.renderer.FillRect(&rect)

		// render countdown
		if notification.close == nil {
			s := strconv.Itoa(int(time.Until(notification.end)/time.Second) + 1)
			sr.printString(s, _notifyCountdownSgr[notification.Type], &types.XY{
				X: windowRect.W - _WIDGET_OUTER_MARGIN - countdownW,
				Y: _WIDGET_OUTER_MARGIN + offset,
			})
		}

		// render text
		sr.printString(notification.Message, notifyColourSgr[notification.Type], &types.XY{
			X: _WIDGET_OUTER_MARGIN + sr.notifyIconSize.X + sr.glyphSize.X,
			Y: _WIDGET_OUTER_MARGIN + offset,
		})

		if surface, ok := sr.notifyIcon[int(notification.Type)].Asset().(*sdl.Surface); ok {
			srcRect := &sdl.Rect{
				X: 0,
				Y: 0,
				W: surface.W,
				H: surface.H,
			}

			dstRect := &sdl.Rect{
				X: _WIDGET_INNER_MARGIN * 2,
				Y: offset + ((textHeight + _WIDGET_OUTER_MARGIN + _WIDGET_OUTER_MARGIN + 2) / 2) - (sr.notifyIconSize.Y / 2),
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

		offset += textHeight + (_WIDGET_INNER_MARGIN * 3)
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

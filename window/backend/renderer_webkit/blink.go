package rendererwebkit

import "time"

func (wr *webkitRender) blinkSlowLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		wr.SetBlinkState(!wr.GetBlinkState())
		wr.TriggerRedraw()
	}
}

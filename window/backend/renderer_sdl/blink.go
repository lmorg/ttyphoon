package rendersdl

import "time"

func (sr *sdlRender) blinkSlowLoop() {
	d := 500 * time.Millisecond

	for {
		select {
		case <-time.After(d):
			sr._blinkSlow.Store(!sr._blinkSlow.Load())
			sr.TriggerRedraw()
		}
	}
}

func (sr *sdlRender) GetBlinkState() bool      { return sr._blinkSlow.Load() }
func (sr *sdlRender) SetBlinkState(state bool) { sr._blinkSlow.Store(state) }

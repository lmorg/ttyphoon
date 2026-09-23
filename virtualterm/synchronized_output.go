package virtualterm

import "time"

const synchronizedUpdateTimeout = time.Second

func (term *Term) beginSynchronizedUpdate(now time.Time) {
	term._synchronizedUpdateDeadline = now.Add(synchronizedUpdateTimeout)
}

func (term *Term) endSynchronizedUpdate() {
	if term._synchronizedUpdateDeadline.IsZero() {
		return
	}

	term._synchronizedUpdateDeadline = time.Time{}
	if term.renderer != nil {
		term.renderer.TriggerRedraw()
	}
}

func (term *Term) synchronizedUpdateActive(now time.Time) bool {
	if term._synchronizedUpdateDeadline.IsZero() {
		return false
	}

	if now.Before(term._synchronizedUpdateDeadline) {
		return true
	}

	term._synchronizedUpdateDeadline = time.Time{}
	return false
}

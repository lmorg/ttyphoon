package rendererwebkit

import (
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

// Parallel sub-agents raise several stickies back to back, so IDs minted from
// the wall clock collide and the frontend collapses them into one.
func TestDisplayStickyIDsAreUnique(t *testing.T) {
	wr := &webkitRender{}

	const count = 8
	seen := make(map[int64]int, count)
	for i := range count {
		nt, ok := wr.DisplaySticky(types.NOTIFY_INFO, "Running subagent", func() {}).(*notificationT)
		if !ok {
			t.Fatal("DisplaySticky did not return *notificationT")
		}
		seen[nt.id]++
		_ = i
	}

	if len(seen) != count {
		t.Fatalf("got %d unique notification IDs for %d stickies, want %d", len(seen), count, count)
	}
}

// delete() matched on ID, so a duplicate ID would evict an unrelated
// notification and leak the real one.
func TestDeleteRemovesTheGivenNotification(t *testing.T) {
	wr := &webkitRender{}

	first := wr.DisplaySticky(types.NOTIFY_INFO, "first", func() {}).(*notificationT)
	second := wr.DisplaySticky(types.NOTIFY_INFO, "second", func() {}).(*notificationT)

	wr.notifications.delete(second)

	wr.notifications.mutex.Lock()
	defer wr.notifications.mutex.Unlock()

	if len(wr.notifications.sticky) != 1 {
		t.Fatalf("sticky count = %d, want 1", len(wr.notifications.sticky))
	}
	if wr.notifications.sticky[0] != first {
		t.Fatal("delete removed the wrong notification")
	}
}

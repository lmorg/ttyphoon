package notes

import (
	"math"
	"sync"
	"testing"
)

func TestLayoutSizingUpdatesAreIndependentAndValidated(t *testing.T) {
	key := t.TempDir()
	var workers sync.WaitGroup
	for name, ratio := range map[string]float64{"fileListRatio": 0.3, "notesRatio": 0.6, "toolsRatio": 0.4} {
		workers.Add(1)
		go func(name string, ratio float64) {
			defer workers.Done()
			if err := setLayoutRatio(key, name, ratio); err != nil {
				t.Error(err)
			}
		}(name, ratio)
	}
	workers.Wait()
	want := LayoutSizing{FileListRatio: 0.3, NotesRatio: 0.6, ToolsRatio: 0.4}
	if got := getLayoutSizing(key); got != want {
		t.Fatalf("layout = %+v, want %+v", got, want)
	}
	for _, ratio := range []float64{-1, 1, math.NaN(), math.Inf(1)} {
		if err := setLayoutRatio(key, "notesRatio", ratio); err == nil {
			t.Fatalf("invalid ratio %v accepted", ratio)
		}
	}
	if err := setLayoutRatio(key, "unknown", 0.5); err == nil {
		t.Fatal("unknown layout field accepted")
	}
	if err := setLayoutRatio(key, "toolsRatio", 0); err != nil {
		t.Fatal(err)
	}
	want.ToolsRatio = 0
	if got := getLayoutSizing(key); got != want {
		t.Fatalf("reset changed unrelated layout sizes: %+v", got)
	}
}

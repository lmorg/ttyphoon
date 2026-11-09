package mainwindow

import (
	"context"
	"reflect"
	"testing"

	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/wailsruntime"
)

type loggingTestRuntime struct {
	types.DesktopRuntime
	event string
	data  []any
}

func (r *loggingTestRuntime) Emit(event string, data ...any) {
	r.event = event
	r.data = data
}

func TestLogWriterEmitsNotesLogThroughDesktopRuntime(t *testing.T) {
	for _, test := range []struct {
		name, input, want string
	}{
		{"default level", "2026/11/09 11:51:00 hello\n", "2026/11/09 11:51:00 [debug] hello"},
		{"explicit level", "2026/11/09 11:51:00 [info] hello\n", "2026/11/09 11:51:00 [info] hello"},
		{"multiline", "2026/11/09 11:51:00 [error] first\nsecond\n", "2026/11/09 11:51:00 [error] first | second"},
	} {
		t.Run(test.name, func(t *testing.T) {
			desktop := &loggingTestRuntime{}
			writer := &logWriter{ctx: wailsruntime.WithRuntime(context.Background(), desktop)}
			n, err := writer.Write([]byte(test.input))
			if err != nil || n != len(test.input) {
				t.Fatalf("Write() = %d, %v; want %d, nil", n, err, len(test.input))
			}
			if desktop.event != "notesLog" || !reflect.DeepEqual(desktop.data, []any{test.want}) {
				t.Fatalf("event = %q, data = %#v; want notesLog, %q", desktop.event, desktop.data, test.want)
			}
		})
	}
}

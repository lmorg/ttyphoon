package tmux

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/lmorg/ttyphoon/config"
)

func TestSessionOSC8CapabilityFollowsConfig(t *testing.T) {
	previous := config.Config.Terminal.OSC8Hyperlinks
	t.Cleanup(func() { config.Config.Terminal.OSC8Hyperlinks = previous })
	for _, enabled := range []bool{false, true} {
		for _, command := range []string{START_NEW_SESSION, START_ATTACH_SESSION} {
			t.Run(fmt.Sprint(enabled)+"/"+command, func(t *testing.T) {
				config.Config.Terminal.OSC8Hyperlinks = enabled
				want := []string{"tmux", "-CC", command}
				if enabled {
					want = []string{"tmux", "-CC", "-T", "hyperlinks", command}
				}
				if got := sessionCommand(command).Args; !reflect.DeepEqual(got, want) {
					t.Fatalf("tmux command = %q, want %q", got, want)
				}
			})
		}
	}
}

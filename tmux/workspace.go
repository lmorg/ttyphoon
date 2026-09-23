package tmux

import (
	"fmt"
	"os"
	"strings"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/workspace"
)

func (tmux *Tmux) WorkspaceStart(name string) {
	cmd := fmt.Sprintf(`new-window -n "%s"`, name)
	cmd += workspaceTermCmdBuilder(config.Config.Workspaces.Workspace(name))

	resp, err := tmux.SendCommand([]byte(cmd))
	if err != nil {
		tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	var msg string
	for _, slice := range resp.Message {
		msg += string(slice) + "\n"
	}
	msg = strings.TrimSpace(msg)

	if resp != nil && len(resp.Message) > 0 {
		icon := types.NOTIFY_INFO
		if resp.IsErr {
			icon = types.NOTIFY_ERROR
		}
		tmux.renderer.DisplayNotification(icon, msg)
	}
}

func workspaceTermCmdBuilder(t *workspace.TermT) string {
	if t == nil {
		return ""
	}

	var cmd string
	if t.Pwd != "" {
		cmd += fmt.Sprintf(` -c "%s"`, os.ExpandEnv(t.Pwd))
	}
	for k, v := range t.Envs {
		cmd += fmt.Sprintf(` -e "%s=%s"`, k, v)
	}
	if t.Cmd != "" {
		cmd += fmt.Sprintf(` "%s"`, t.Cmd)
	}
	return cmd
}

/*
tmux -C new-session -d -s build <<'EOF'
new-window -n services -c ~/src/api -e LOG_LEVEL=debug 'npm run dev'
split-window -h -c ~/src/web -e PORT=3000 'npm run start'
split-window -v -c ~/src/worker -e QUEUE=default
send-keys -t %2 'make watch' Enter
select-layout tiled
refresh-client -f no-output
detach-client
EOF
*/

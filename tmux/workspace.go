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
	settings := config.Config.Workspaces.Workspace(name)
	workingDirectory := ""
	if settings.Pwd != "" {
		workingDirectory = os.ExpandEnv(settings.Pwd)
	} else if pane, err := tmux.ActivePane(); err == nil && pane != nil {
		workingDirectory = pane.Pwd()
	}
	environment := cloneEnvironment(tmux.initialEnvVars)
	for key, value := range settings.Envs {
		environment[key] = value
	}
	baseEnvironment := cloneEnvironment(tmux.initialEnvVars)
	for key := range settings.Envs {
		delete(baseEnvironment, key)
	}
	cmd := fmt.Sprintf(`new-window -n "%s" -P -F '#{pane_id}'`, name)
	if workingDirectory != "" {
		cmd += " -c " + workspace.QuoteString(workingDirectory)
	}
	cmd += workspace.EnvsTmux(baseEnvironment)
	cmd += workspaceTermCmdBuilder(&workspace.TermT{Envs: settings.Envs, Cmd: settings.Cmd})

	resp, err := tmux.SendCommand([]byte(cmd))
	if err != nil {
		tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}
	paneID := responsePaneID(resp)
	tmux.seedPaneStartupState(paneID, workingDirectory, environment)

	var msg string
	for _, slice := range resp.Message {
		line := strings.TrimSpace(string(slice))
		if line != "" && line != paneID {
			msg += line + "\n"
		}
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

package tmux

import (
	"fmt"
	"os"
	"strings"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/envvars"
	"github.com/lmorg/ttyphoon/utils/workspace"
)

func (tmux *Tmux) WorkspaceStart(name string) {
	settings := config.Config.Workspaces.Workspace(name)
	scoped, err := envvars.Resolve(settings.EnvFiles, settings.Envs)
	if err != nil {
		tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("Cannot load workspace environment: %v", err))
		return
	}
	if len(settings.EnvFiles) > 0 || len(settings.Envs) > 0 {
		tmux.privateEnvironment.Store(true)
	}
	workingDirectory := ""
	if settings.Pwd != "" {
		workingDirectory = os.Expand(settings.Pwd, func(name string) string {
			value, _ := scoped.Lookup(name)
			return value
		})
	} else if pane, err := tmux.ActivePane(); err == nil && pane != nil {
		workingDirectory = pane.Pwd()
	}
	environment := scoped.Overlay(tmux.initialEnvVars)
	cmd := fmt.Sprintf(`new-window -n %s -P -F '#{pane_id}'`, workspace.QuoteString(name))
	if workingDirectory != "" {
		cmd += " -c " + workspace.QuoteString(workingDirectory)
	}
	cmd += workspace.EnvsTmux(environment)
	if settings.Cmd != "" {
		cmd += " " + workspace.QuoteString(settings.Cmd)
	}

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

	if msg != "" {
		icon := types.NOTIFY_INFO
		if resp.IsErr {
			icon = types.NOTIFY_ERROR
		}
		tmux.renderer.DisplayNotification(icon, msg)
	}
}

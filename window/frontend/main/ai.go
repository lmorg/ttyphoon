package mainwindow

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/ai"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/sessiondb"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
)

func (a *WApp) RunAIAgentWithStream(tileID, prompt string) error {
	agt := agent.Get(tileID)
	if agt == nil {
		return fmt.Errorf("agent not found for tile %s", tileID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	_, err := agt.RunLLMWithStream(ctx, prompt, nil)

	return err
}

// ListAIPromptLogs returns SQLite prompt metadata for the active AI session,
// ordered chronologically (oldest first).
func (a *WApp) ListAIPromptLogs() []sessiondb.PromptLogMeta {
	agt, ok := a.activeAgent()
	if !ok {
		return nil
	}
	return sessiondb.ListPromptLogs(agt.Workspace())
}

func (a *WApp) GetAISessionPrompt(sessionID, promptID int64) string {
	agt, ok := a.activeAgent()
	if !ok {
		return ""
	}
	prompt, err := sessiondb.GetSessionPrompt(agt.Workspace(), sessionID, promptID)
	if err != nil {
		log.Printf("[debug] ai get session prompt %d: %v", promptID, err)
		return ""
	}
	return prompt
}

func (a *WApp) ListAIStreamBlocks(sessionID, promptID int64) []sessiondb.StreamBlockMeta {
	agt, ok := a.activeAgent()
	if !ok {
		return nil
	}
	blocks, err := sessiondb.ListStreamBlockMeta(agt.Workspace(), sessionID, promptID)
	if err != nil {
		log.Printf("[debug] ai list stream blocks: %v", err)
		return nil
	}
	return blocks
}

// ListAILiveStreamBlocks returns the blocks backing the panel's live view: the
// in-flight run when one is open, otherwise the newest finalized prompt.
func (a *WApp) ListAILiveStreamBlocks(workspace string) []sessiondb.StreamBlockMeta {
	if strings.TrimSpace(workspace) == "" {
		agt, ok := a.activeAgent()
		if !ok {
			return nil
		}
		workspace = agt.Workspace()
	}
	blocks, err := sessiondb.ListLiveStreamBlockMeta(workspace)
	if err != nil {
		log.Printf("[debug] ai list live stream blocks: %v", err)
		return nil
	}
	return blocks
}

func (a *WApp) GetAIStreamBlockContent(sessionID, promptID int64, blockID string) string {
	agt, ok := a.activeAgent()
	if !ok {
		return ""
	}
	content, err := sessiondb.GetStreamBlockContent(agt.Workspace(), sessionID, promptID, blockID)
	if err != nil {
		log.Printf("[debug] ai get stream block %q: %v", blockID, err)
		return ""
	}
	return content
}

func (a *WApp) activeAgent() (*agent.Agent, bool) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok || renderer == nil {
		return nil, false
	}

	tile := renderer.ActiveTile()
	if tile == nil {
		return nil, false
	}

	agt, ok := agent.TryGet(tile.Id())
	if !ok || agt == nil {
		return nil, false
	}

	return agt, true
}

func (a *WApp) ListAIModelSelections() []string {
	agt, ok := a.activeAgent()
	if !ok {
		return []string{}
	}

	_, labels := agt.ListModels()
	slices.Sort(labels)
	return labels
}

func (a *WApp) GetCurrentAIModelSelection() string {
	agt, ok := a.activeAgent()
	if !ok {
		return ""
	}

	return agt.CurrentModelLabel()
}

// SetAIPanelLive tells the backend whether the AI panel is following live output
// or showing a historical prompt, so background runs stop emitting to the UI.
func (a *WApp) SetAIPanelLive(workspace string, live bool) {
	if strings.TrimSpace(workspace) == "" {
		agt, ok := a.activeAgent()
		if !ok {
			return
		}
		workspace = agt.Workspace()
	}

	sessiondb.SetPanelView(workspace, live)
}

func (a *WApp) GetAIExecutionLimits() map[string]any {
	agt, ok := a.activeAgent()
	if !ok {
		return map[string]any{}
	}

	return map[string]any{
		"agentSteps":            agt.MaxIterations(),
		"providerMaxIterations": nil,
		"modelContextWindow":    nil,
		"modelMaxOutputTokens":  nil,
		"requestTimeout":        config.Config.Ai.RequestTimeoutDuration().String(),
	}
}

func (a *WApp) SetCurrentAIModelSelection(selection string) error {
	agt, ok := a.activeAgent()
	if !ok {
		return errors.New("AI agent is unavailable")
	}

	err := agt.SetServiceModelFromSelection(selection)
	if err != nil {
		log.Printf("ai: failed to set model selection %q: %v", selection, err)
		return err
	}

	return nil
}

func (a *WApp) ShowAIToolsMenu() {
	agt, ok := a.activeAgent()
	if !ok {
		return
	}

	agt.ChooseTools(nil)
}

func (a *WApp) ShowAISkillsMenu(x, y float64) {
	renderwebkit.AskAiSkillsAt(int(x), int(y))
}

func (a *WApp) GetAIToolsList() []map[string]interface{} {
	agt, ok := a.activeAgent()
	if !ok {
		return []map[string]interface{}{}
	}

	return agt.ListTools()
}

func (a *WApp) SetAIToolEnabled(toolName string, enabled bool) error {
	agt, ok := a.activeAgent()
	if !ok {
		return fmt.Errorf("AI agent is unavailable")
	}

	return agt.SetToolEnabled(toolName, enabled)
}

func (a *WApp) SetAIToolSubagentAllowed(toolName string, allowed bool) error {
	agt, ok := a.activeAgent()
	if !ok {
		return fmt.Errorf("AI agent is unavailable")
	}
	return agt.SetToolAllowedInSubagent(toolName, allowed)
}

func (a *WApp) ShowAIToolSubagentMenu(toolName string, x, y float64) {
	agt, ok := a.activeAgent()
	if !ok {
		return
	}

	agt.ShowToolSubagentMenu(toolName, int(x), int(y), func(allowed bool) {
		runtime.EventsEmit(a.ctx, "aiToolStateChanged", map[string]any{
			"name":            toolName,
			"allowInSubagent": allowed,
		})
	})
}

func (a *WApp) SetAIToolState(toolName, state string) error {
	agt, ok := a.activeAgent()
	if !ok {
		return fmt.Errorf("AI agent is unavailable")
	}
	return agt.SetToolState(toolName, state)
}

func (a *WApp) ShowAIToolStateMenu(toolName string, x, y float64) {
	agt, ok := a.activeAgent()
	if !ok {
		return
	}
	agt.ShowToolStateMenu(toolName, int(x), int(y), func(state string) {
		runtime.EventsEmit(a.ctx, "aiToolStateChanged", map[string]string{
			"name":  toolName,
			"state": state,
		})
	})
}

// ResolveAIToolPermission is called when the user clicks an access-request
// option rendered in the AI panel output (see agent.RequestWritePermission).
func (a *WApp) ResolveAIToolPermission(requestID, decision string) error {
	if decision != "allow-once" && decision != "allow-prompt" {
		return fmt.Errorf("tool denial decisions must use the reason inputbox")
	}
	return agent.ResolveWritePermissionRequest(requestID, decision, "")
}

func (a *WApp) PromptAIToolPermissionReason(requestID, decision string) error {
	if decision != "deny-once" && decision != "deny-prompt" {
		return fmt.Errorf("unsupported tool denial decision %q", decision)
	}
	workspace, toolName, ok := agent.WritePermissionRequestDetails(requestID)
	if !ok {
		return fmt.Errorf("permission request %q not found or already resolved", requestID)
	}
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return fmt.Errorf("renderer is unavailable")
	}
	denialScope := "this invocation"
	confirmLabel := "Deny invocation"
	if decision == "deny-prompt" {
		denialScope = "this prompt"
		confirmLabel = "Deny prompt"
	}
	renderer.DisplayInputBoxW(&types.InputBoxWT{
		Options: types.InputBoxWTOptions{
			Title:        fmt.Sprintf("Why deny %s for %s?", toolName, denialScope),
			Placeholder:  "Optional reason to share with the AI",
			Multiline:    true,
			CacheKey:     "ai-tool-denial-reasons:" + workspace,
			DraftKey:     "ai-tool-denial:" + workspace + ":" + requestID,
			ConfirmLabel: confirmLabel,
			CancelLabel:  "Cancel",
		},
		OkFunc: func(value *types.InputBoxCallbackResultT) {
			if err := agent.ResolveWritePermissionRequest(requestID, decision, value.String()); err != nil {
				runtime.EventsEmit(a.ctx, "aiToolPermissionResolutionError", map[string]string{
					"requestId": requestID,
					"error":     err.Error(),
				})
				return
			}
			runtime.EventsEmit(a.ctx, "aiToolPermissionResolved", map[string]string{
				"requestId": requestID,
				"decision":  decision,
			})
		},
	})
	return nil
}

func (a *WApp) ResolveAIUserQuestion(requestID, answer string) error {
	return agent.ResolveUserQuestionRequest(requestID, answer)
}

func (a *WApp) GetAIMcpServers() []map[string]any {
	agt, ok := a.activeAgent()
	if !ok {
		return []map[string]any{}
	}

	return agt.ListMcpServers()
}

func (a *WApp) SetAIMcpServerEnabled(serverKey string, enabled bool) error {
	agt, ok := a.activeAgent()
	if !ok {
		return fmt.Errorf("AI agent is unavailable")
	}

	return agt.SetMcpServerEnabled(serverKey, enabled)
}

func (a *WApp) ShowAIMcpMenu() {
	agt, ok := a.activeAgent()
	if !ok {
		return
	}

	agt.McpMenu(nil)
}

func (a *WApp) GetAISessionManagement() sessiondb.FrontendStateT {
	agt, ok := a.activeAgent()
	if !ok {
		return sessiondb.FrontendStateT{}
	}

	state, err := sessiondb.GetFrontendState(agt.Workspace(), 24)
	if err != nil {
		log.Printf("[debug] ai session management: %v", err)
		return sessiondb.FrontendStateT{}
	}

	return state
}

func (a *WApp) CreateAISession(summary string) sessiondb.FrontendStateT {
	agt, ok := a.activeAgent()
	if !ok {
		return sessiondb.FrontendStateT{}
	}

	state, err := sessiondb.CreateSession(agt.Workspace(), summary, 24)
	if err != nil {
		log.Printf("[debug] ai create session: %v", err)
		return sessiondb.FrontendStateT{}
	}

	return state
}

func (a *WApp) SetActiveAISession(tableID int64) sessiondb.FrontendStateT {
	agt, ok := a.activeAgent()
	if !ok {
		return sessiondb.FrontendStateT{}
	}

	state, err := sessiondb.SetActiveSession(agt.Workspace(), tableID, 24)
	if err != nil {
		log.Printf("[debug] ai set active session: %v", err)
		return sessiondb.FrontendStateT{}
	}

	return state
}

func (a *WApp) DeleteAISession(tableID int64) sessiondb.FrontendStateT {
	agt, ok := a.activeAgent()
	if !ok {
		return sessiondb.FrontendStateT{}
	}

	state, err := sessiondb.DeleteSession(agt.Workspace(), tableID, 24)
	if err != nil {
		log.Printf("ai delete session: %v", err)
		return sessiondb.FrontendStateT{}
	}

	if err := sessiondb.DeleteSessionLog(agt.Workspace(), tableID); err != nil {
		log.Printf("[debug] [debug] ai delete session log: %v", err)
	}

	return state
}

func (a *WApp) RenameAISession(tableID int64, summary string) sessiondb.FrontendStateT {
	agt, ok := a.activeAgent()
	if !ok {
		return sessiondb.FrontendStateT{}
	}

	state, err := sessiondb.RenameSession(agt.Workspace(), tableID, summary, 24)
	if err != nil {
		log.Printf("ai rename session: %v", err)
		return sessiondb.FrontendStateT{}
	}

	return state
}

func (a *WApp) ClearAISessionHistory() sessiondb.FrontendStateT {
	agt, ok := a.activeAgent()
	if !ok {
		return sessiondb.FrontendStateT{}
	}

	state, err := sessiondb.ClearActiveSession(agt.Workspace(), 24)
	if err != nil {
		log.Printf("[debug] ai clear session history: %v", err)
		return sessiondb.FrontendStateT{}
	}

	if err := sessiondb.ClearActiveSessionLog(agt.Workspace()); err != nil {
		log.Printf("[debug] ai clear session log: %v", err)
	}

	return state
}

func (a *WApp) ClearAILog() {
	agt, ok := a.activeAgent()
	if !ok {
		return
	}

	if err := sessiondb.ClearActiveSessionLog(agt.Workspace()); err != nil {
		log.Printf("[debug] ai clear log: %v", err)
	}
}

// DeleteAIHistoryEntry removes a single prompt/response entry from the active
// session, both the sqlite row shown in the Settings transcript and its
// per-prompt markdown log file.
func (a *WApp) DeleteAIHistoryEntry(entryID int64) sessiondb.FrontendStateT {
	agt, ok := a.activeAgent()
	if !ok {
		return sessiondb.FrontendStateT{}
	}

	state, err := sessiondb.DeleteActiveSessionEntry(agt.Workspace(), entryID, 24)
	if err != nil {
		log.Printf("[debug] ai delete history entry: %v", err)
		return sessiondb.FrontendStateT{}
	}

	if err := sessiondb.DeletePromptLog(agt.Workspace(), state.ActiveSessionID, entryID); err != nil {
		log.Printf("[debug] ai delete prompt log: %v", err)
	}

	return state
}

func (a *WApp) AskAI(callerType, filename, contents string) {
	log.Printf(`[debug] WApp AskAI: callerType="%s"`, callerType)

	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	switch callerType {
	case "notesDocument":
		tile := renderer.ActiveTile()
		if tile == nil {
			return
		}

		agt := agent.Get(tile.Id())
		if agt == nil {
			return
		}
		ai.ExplainDoc(agt, filename, contents, a.collectAIImageAttachments(contents))
	case "notesPromptToolbar":
		renderer.AskAi()
	case "notesPromptUri":
		tile := renderer.ActiveTile()
		if tile == nil {
			return
		}
		agt := agent.Get(tile.Id())
		if agt == nil {
			return
		}
		ai.UriPrompt(agt, filename, contents)
	default:
		return
	}
}

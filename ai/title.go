package ai

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/prompts"
	"github.com/lmorg/ttyphoon/types"
)

const aiNoteTitleTimeout = 30 * time.Second
const aiNoteTitleMaxLen = 64

var rxAINoteTitleSep = regexp.MustCompile(`[^a-z0-9]+`)

func sanitizeAINoteTitle(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	title = rxAINoteTitleSep.ReplaceAllString(title, "-")
	title = strings.Trim(title, "-")

	if len(title) > aiNoteTitleMaxLen {
		title = strings.Trim(title[:aiNoteTitleMaxLen], "-")
	}

	return title
}

func fallbackAINoteTitle(agent *agent.Agent) string {
	parts := make([]string, 0, 2)

	if s := sanitizeAINoteTitle(agent.Meta.Function); s != "" {
		parts = append(parts, s)
	}

	if s := sanitizeAINoteTitle(agent.ServiceName()); s != "" {
		parts = append(parts, s)
	}

	if len(parts) == 0 {
		return "ai-note"
	}

	return strings.Join(parts, "-")
}

func summarizeAINoteTitle(agent *agent.Agent, query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), aiNoteTitleTimeout)
	defer cancel()

	title, err := agent.RunLLMWithStream(ctx, prompts.GetTitle(agent, query), nil)
	if err != nil {
		return ""
	}

	return sanitizeAINoteTitle(title)
}

func buildAINoteFilename(agent *agent.Agent, query string, now time.Time) string {
	sticky := agent.Renderer().DisplaySticky(types.NOTIFY_INFO, "Writing output to markdown...", nil)
	defer sticky.Close()

	title := summarizeAINoteTitle(agent, query)
	if title == "" {
		title = fallbackAINoteTitle(agent)
	}

	return fmt.Sprintf("$GLOBAL/ai-%s-%d.md", title, now.Unix())
}

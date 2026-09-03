package agent

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
)

var userQuestionRequests = struct {
	mu sync.Mutex
	m  map[string]chan string
}{m: map[string]chan string{}}

var userQuestionSeq int64

func newUserQuestionRequest() (string, chan string) {
	id := fmt.Sprintf("uq%d", atomic.AddInt64(&userQuestionSeq, 1))
	ch := make(chan string, 1)
	userQuestionRequests.mu.Lock()
	userQuestionRequests.m[id] = ch
	userQuestionRequests.mu.Unlock()
	return id, ch
}

func takeUserQuestionRequest(id string) (chan string, bool) {
	userQuestionRequests.mu.Lock()
	defer userQuestionRequests.mu.Unlock()
	ch, ok := userQuestionRequests.m[id]
	if ok {
		delete(userQuestionRequests.m, id)
	}
	return ch, ok
}

func RequestUserQuestion(ctx context.Context, question string, choices []string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(question) == "" {
		return "", fmt.Errorf("question is required")
	}

	requestID, decisionCh := newUserQuestionRequest()
	emitAIStreamToolProgress(ctx, formatUserQuestionRequestMarkdown(question, requestID, choices))

	select {
	case answer := <-decisionCh:
		if strings.TrimSpace(answer) == "" {
			return "", fmt.Errorf("no answer received")
		}
		return answer, nil
	case <-ctx.Done():
		takeUserQuestionRequest(requestID)
		return "", ctx.Err()
	}
}

func ResolveUserQuestionRequest(id, answer string) error {
	ch, ok := takeUserQuestionRequest(id)
	if !ok {
		return nil
	}
	if strings.TrimSpace(answer) == "" {
		close(ch)
		return nil
	}
	ch <- answer
	return nil
}

func formatUserQuestionRequestMarkdown(question, requestID string, choices []string) string {
	question = strings.TrimSpace(question)
	if question == "" {
		question = "Question"
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("\n\n**Question**\n\n%s", question))
	if len(choices) == 0 {
		lines = append(lines, fmt.Sprintf("\n\n- [Answer](ttyphoon://ai-user-question?request=%s&answer=%s)", requestID, url.QueryEscape("answer")))
		return strings.Join(lines, "\n") + "\n\n"
	}

	for _, choice := range choices {
		label := strings.TrimSpace(choice)
		if label == "" {
			continue
		}
		encoded := url.QueryEscape(label)
		lines = append(lines, fmt.Sprintf("- [%s](ttyphoon://ai-user-question?request=%s&answer=%s)", label, requestID, encoded))
	}
	return strings.Join(lines, "\n") + "\n\n"
}

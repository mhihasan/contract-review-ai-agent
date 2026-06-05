package agent

import (
	"context"
	"fmt"
	"log"

	"github.com/mhihasan/contract-review-ai-agent/llm"
	"github.com/mhihasan/contract-review-ai-agent/tokens"
)

const maxToolResultRunes = 500

type ContextManager struct {
	model       string
	windowLimit int
	compactAt   int
	keepRecent  int
	summarizer  llm.LLM
}

func NewContextManager(model string, window int, compactRatio float64, keepRecent int, summarizer llm.LLM) *ContextManager {
	return &ContextManager{
		model:       model,
		windowLimit: window,
		compactAt:   int(float64(window) * compactRatio),
		keepRecent:  keepRecent,
		summarizer:  summarizer,
	}
}

func (cm *ContextManager) Prepare(ctx context.Context, msgs []llm.Message) ([]llm.Message, error) {
	if tokens.CountMessages(cm.model, msgs) < cm.compactAt {
		return msgs, nil
	}

	pinned, middle, recent := cm.split(msgs)

	middle = cm.truncateToolResults(middle)

	result := flatten(pinned, middle, recent)
	if tokens.CountMessages(cm.model, result) < cm.compactAt {
		return result, nil
	}

	summary, err := cm.summarize(ctx, middle)
	if err != nil {
		log.Printf("warn: summarize failed (%v); dropping middle block", err)
		result = flatten(pinned, nil, recent)
		return result, nil
	}
	result = flatten(pinned, []llm.Message{summary}, recent)

	if tokens.CountMessages(cm.model, result) > cm.windowLimit {
		log.Printf("warn: context still over window (%d > %d) after compaction; dropping oldest summarized content",
			tokens.CountMessages(cm.model, result), cm.windowLimit)
		result = flatten(pinned, nil, recent)
	}

	return result, nil
}

func (cm *ContextManager) split(msgs []llm.Message) (pinned, middle, recent []llm.Message) {
	if len(msgs) == 0 {
		return nil, nil, nil
	}

	pinned = msgs[:1]

	recentStart := len(msgs) - cm.keepRecent
	if recentStart <= 1 {
		recentStart = 1
	}

	middle = msgs[1:recentStart]
	recent = msgs[recentStart:]
	return pinned, middle, recent
}

func (cm *ContextManager) truncateToolResults(msgs []llm.Message) []llm.Message {
	out := make([]llm.Message, len(msgs))
	copy(out, msgs)
	for i, m := range out {
		if m.Role == llm.RoleTool && len([]rune(m.Content)) > maxToolResultRunes {
			runes := []rune(m.Content)
			head := string(runes[:maxToolResultRunes/2])
			tail := string(runes[len(runes)-maxToolResultRunes/2:])
			out[i].Content = head + "\n...[truncated]...\n" + tail
		}
	}
	return out
}

func (cm *ContextManager) summarize(ctx context.Context, msgs []llm.Message) (llm.Message, error) {
	var sb string
	for _, m := range msgs {
		sb += string(m.Role) + ": " + m.Content + "\n"
	}
	prompt := "Summarize these prior steps and findings concisely so analysis can continue:\n\n" + sb

	resp, err := cm.summarizer.Complete(ctx, llm.CompletionRequest{
		Messages:  []llm.Message{{Role: llm.RoleUser, Content: prompt}},
		MaxTokens: 512,
	})
	if err != nil {
		return llm.Message{}, fmt.Errorf("summarize: %w", err)
	}
	return llm.Message{
		Role:    llm.RoleAssistant,
		Content: resp.Content,
	}, nil
}

func flatten(pinned, middle, recent []llm.Message) []llm.Message {
	out := make([]llm.Message, 0, len(pinned)+len(middle)+len(recent))
	out = append(out, pinned...)
	out = append(out, middle...)
	out = append(out, recent...)
	return out
}

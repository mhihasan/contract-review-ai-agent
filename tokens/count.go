package tokens

import (
	"github.com/mhihasan/contract-review-ai-agent/llm"
	tiktoken "github.com/pkoukk/tiktoken-go"
)

const perMessageOverhead = 4

func Count(model string, text string) int {
	enc, err := tiktoken.EncodingForModel(model)
	if err != nil {
		enc, err = tiktoken.GetEncoding("cl100k_base")
		if err != nil {
			return len([]rune(text)) / 4
		}
	}
	return len(enc.Encode(text, nil, nil))
}

func CountMessages(model string, msgs []llm.Message) int {
	total := 0
	for _, m := range msgs {
		total += Count(model, string(m.Role))
		total += Count(model, m.Content)
		for _, tc := range m.ToolCalls {
			total += Count(model, tc.Name)
			total += Count(model, string(tc.Args))
		}
		if m.ToolCallID != "" {
			total += Count(model, m.ToolCallID)
		}
		total += perMessageOverhead
	}
	return total
}

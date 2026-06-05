package agent_test

import (
	"testing"

	"github.com/mhihasan/contract-review-ai-agent/agent"
)

func TestAgentRunStatusConstants(t *testing.T) {
	statuses := []string{
		agent.AgentRunStatusRunning,
		agent.AgentRunStatusSubmitted,
		agent.AgentRunStatusMaxSteps,
		agent.AgentRunStatusBudget,
		agent.AgentRunStatusFailed,
	}
	for _, s := range statuses {
		if s == "" {
			t.Errorf("status constant must not be empty")
		}
	}
}

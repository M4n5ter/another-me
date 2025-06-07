package orchestrator

import "github.com/m4n5ter/another-me/pkg/llminterface"

type Orchestrator struct {
	llm llminterface.ChatAdapter
}

func NewOrchestrator(llm llminterface.ChatAdapter) *Orchestrator {
	return &Orchestrator{
		llm: llm,
	}
}

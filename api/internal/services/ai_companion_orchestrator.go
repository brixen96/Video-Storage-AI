package services

import (
	"context"
)

// LLMOrchestrator is a lightweight orchestrator for LLM-driven conversations.
// This file provides a safe, compile-friendly skeleton to be expanded in the next steps.
type LLMOrchestrator struct {
	// attach references as needed, e.g. db, tool registry, logger
}

// NewLLMOrchestrator creates a new orchestrator instance.
func NewLLMOrchestrator() *LLMOrchestrator {
	return &LLMOrchestrator{}
}

// Orchestrate runs a single conversation request. Implement function-calling and streaming here.
func (o *LLMOrchestrator) Orchestrate(ctx context.Context, input string) error {
	// TODO: call LLM, handle tool_call responses, stream results
	return nil
}

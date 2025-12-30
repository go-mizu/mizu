package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/cms/pkg/llm"
	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

// Orchestrator coordinates multiple agents.
type Orchestrator struct {
	registry *Registry
	llm      llm.Client
	store    Store
}

// NewOrchestrator creates a new orchestrator.
func NewOrchestrator(registry *Registry, llmClient llm.Client, store Store) *Orchestrator {
	return &Orchestrator{
		registry: registry,
		llm:      llmClient,
		store:    store,
	}
}

// Route determines which agent(s) should handle a request.
func (o *Orchestrator) Route(ctx context.Context, input string) ([]AgentType, error) {
	if o.llm == nil {
		return nil, fmt.Errorf("LLM client not configured")
	}

	// Build agent descriptions for routing
	agents := o.registry.List()
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents registered")
	}

	var agentDescriptions strings.Builder
	for _, agent := range agents {
		agentDescriptions.WriteString(fmt.Sprintf("- %s: %s\n", agent.Type, agent.Description))
		for _, cap := range agent.Capabilities {
			agentDescriptions.WriteString(fmt.Sprintf("  - %s: %s\n", cap.Action, cap.Description))
		}
	}

	prompt := fmt.Sprintf(`You are an AI agent router. Based on the user's request, determine which agent(s) should handle it.

Available agents:
%s

User request: %s

Respond with a JSON array of agent types that should handle this request, in order of execution.
For example: ["content", "seo"]

Only include agents that are directly relevant to the request.`, agentDescriptions.String(), input)

	resp, err := o.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: prompt},
		},
		MaxTokens:   100,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM routing failed: %w", err)
	}

	// Parse response
	content := strings.TrimSpace(resp.Content)
	// Extract JSON array from response
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start == -1 || end == -1 || start > end {
		return nil, fmt.Errorf("invalid routing response: %s", content)
	}
	content = content[start : end+1]

	var agentTypes []string
	if err := json.Unmarshal([]byte(content), &agentTypes); err != nil {
		return nil, fmt.Errorf("parse routing response: %w", err)
	}

	result := make([]AgentType, len(agentTypes))
	for i, t := range agentTypes {
		result[i] = AgentType(t)
	}

	return result, nil
}

// Execute runs an action with the appropriate agent.
func (o *Orchestrator) Execute(ctx context.Context, agentType AgentType, input *ActionInput) (*ActionOutput, error) {
	agent, ok := o.registry.Get(agentType)
	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentType)
	}

	if !agent.CanHandle(input.Action) {
		return nil, fmt.Errorf("agent %s cannot handle action %s", agentType, input.Action)
	}

	// Create task record
	task := &Task{
		ID:        ulid.New(),
		AgentType: agentType,
		Action:    input.Action,
		Status:    TaskStatusRunning,
		Input:     input.Data,
		CreatedAt: time.Now(),
	}
	now := time.Now()
	task.StartedAt = &now

	if o.store != nil {
		if err := o.store.CreateTask(ctx, task); err != nil {
			// Log error but continue execution
			fmt.Printf("failed to create task: %v\n", err)
		}
	}

	// Execute agent action
	output, err := agent.Execute(ctx, input.Action, input)

	// Update task
	completedAt := time.Now()
	task.CompletedAt = &completedAt

	if err != nil {
		task.Status = TaskStatusFailed
		task.Error = err.Error()
	} else {
		task.Status = TaskStatusCompleted
		task.Output = output.Data
	}

	if o.store != nil {
		if updateErr := o.store.UpdateTask(ctx, task); updateErr != nil {
			fmt.Printf("failed to update task: %v\n", updateErr)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	return output, nil
}

// ExecuteNL processes a natural language command.
func (o *Orchestrator) ExecuteNL(ctx context.Context, command string, agentCtx *AgentContext) (*ActionOutput, error) {
	// Route to appropriate agent(s)
	agentTypes, err := o.Route(ctx, command)
	if err != nil {
		return nil, fmt.Errorf("routing failed: %w", err)
	}

	if len(agentTypes) == 0 {
		return &ActionOutput{
			Success: false,
			Messages: []Message{
				{Role: "assistant", Content: "I'm not sure how to handle that request. Could you please be more specific?"},
			},
		}, nil
	}

	// Parse the command to extract action and parameters
	parsedInput, err := o.parseNLCommand(ctx, command, agentTypes[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse command: %w", err)
	}

	parsedInput.Context = agentCtx

	// Execute with the primary agent
	return o.Execute(ctx, agentTypes[0], parsedInput)
}

// parseNLCommand parses a natural language command into structured input.
func (o *Orchestrator) parseNLCommand(ctx context.Context, command string, agentType AgentType) (*ActionInput, error) {
	agent, ok := o.registry.Get(agentType)
	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentType)
	}

	// Build capabilities description
	var capsDesc strings.Builder
	for _, cap := range agent.Capabilities() {
		capsDesc.WriteString(fmt.Sprintf("- %s: %s\n", cap.Action, cap.Description))
	}

	prompt := fmt.Sprintf(`Parse the following user command into a structured action for the %s agent.

Available actions:
%s

User command: %s

Respond with JSON in this format:
{
  "action": "action_name",
  "collection": "collection_name_if_applicable",
  "documentId": "document_id_if_applicable",
  "data": {
    "key": "value"
  }
}

Only include fields that are relevant to the command.`, agentType, capsDesc.String(), command)

	resp, err := o.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: prompt},
		},
		MaxTokens:   500,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM parsing failed: %w", err)
	}

	// Parse JSON from response
	content := strings.TrimSpace(resp.Content)
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || start > end {
		return nil, fmt.Errorf("invalid parsing response: %s", content)
	}
	content = content[start : end+1]

	var input ActionInput
	if err := json.Unmarshal([]byte(content), &input); err != nil {
		return nil, fmt.Errorf("parse input: %w", err)
	}

	return &input, nil
}

// ExecutePlan executes a multi-step plan.
func (o *Orchestrator) ExecutePlan(ctx context.Context, plan *Plan) (*PlanResult, error) {
	if plan == nil || len(plan.Steps) == 0 {
		return nil, fmt.Errorf("plan is empty")
	}

	startTime := time.Now()
	result := &PlanResult{
		PlanID: plan.ID,
		Steps:  make([]StepResult, len(plan.Steps)),
	}

	// Track completed steps and their outputs
	completed := make(map[string]*ActionOutput)

	for i, step := range plan.Steps {
		stepStart := time.Now()
		stepResult := StepResult{
			StepID: step.ID,
		}

		// Check dependencies
		for _, depID := range step.DependsOn {
			if _, ok := completed[depID]; !ok {
				stepResult.Success = false
				stepResult.Error = fmt.Sprintf("dependency %s not completed", depID)
				stepResult.Duration = time.Since(stepStart)
				result.Steps[i] = stepResult
				continue
			}
		}

		// Execute step
		input := &ActionInput{
			Action: step.Action,
			Data:   step.Input,
		}

		output, err := o.Execute(ctx, step.Agent, input)
		stepResult.Duration = time.Since(stepStart)

		if err != nil {
			stepResult.Success = false
			stepResult.Error = err.Error()
		} else {
			stepResult.Success = true
			stepResult.Output = output
			completed[step.ID] = output
		}

		result.Steps[i] = stepResult
	}

	result.TotalTime = time.Since(startTime)

	// Check overall success
	result.Success = true
	for _, step := range result.Steps {
		if !step.Success {
			result.Success = false
			break
		}
	}

	return result, nil
}

// GetAgent returns an agent by type.
func (o *Orchestrator) GetAgent(agentType AgentType) (Agent, bool) {
	return o.registry.Get(agentType)
}

// ListAgents returns all registered agents.
func (o *Orchestrator) ListAgents() []AgentInfo {
	return o.registry.List()
}

// Package types provides built-in agent implementations.
package types

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-mizu/blueprints/cms/feature/agents"
	"github.com/go-mizu/blueprints/cms/feature/collections"
	"github.com/go-mizu/blueprints/cms/pkg/llm"
)

// Content agent actions
const (
	ContentActionGenerate           = "generate"
	ContentActionRewrite            = "rewrite"
	ContentActionSummarize          = "summarize"
	ContentActionExpand             = "expand"
	ContentActionEdit               = "edit"
	ContentActionProofread          = "proofread"
	ContentActionToneAdjust         = "tone_adjust"
	ContentActionExtractKeywords    = "extract_keywords"
	ContentActionGenerateOutline    = "generate_outline"
	ContentActionGenerateVariations = "generate_variations"
)

// ContentAgent handles content generation and editing.
type ContentAgent struct {
	llm         llm.Client
	collections collections.API
}

// NewContentAgent creates a new content agent.
func NewContentAgent(llmClient llm.Client, collectionsAPI collections.API) *ContentAgent {
	return &ContentAgent{
		llm:         llmClient,
		collections: collectionsAPI,
	}
}

// Type returns the agent type.
func (a *ContentAgent) Type() agents.AgentType {
	return agents.AgentTypeContent
}

// Name returns the agent name.
func (a *ContentAgent) Name() string {
	return "Content Agent"
}

// Description returns what this agent does.
func (a *ContentAgent) Description() string {
	return "Generates, edits, and enhances content using AI. Handles writing, rewriting, summarization, and content optimization."
}

// Capabilities returns the list of actions this agent can perform.
func (a *ContentAgent) Capabilities() []agents.Capability {
	return []agents.Capability{
		{
			Action:      ContentActionGenerate,
			Description: "Generate new content from a prompt",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"prompt":      map[string]string{"type": "string", "description": "What to generate"},
					"collection":  map[string]string{"type": "string", "description": "Target collection"},
					"fields":      map[string]any{"type": "array", "items": map[string]string{"type": "string"}},
					"constraints": map[string]string{"type": "object"},
				},
				"required": []string{"prompt"},
			},
		},
		{
			Action:      ContentActionRewrite,
			Description: "Rewrite existing content with improvements",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content":      map[string]string{"type": "string"},
					"instructions": map[string]string{"type": "string"},
				},
				"required": []string{"content"},
			},
		},
		{
			Action:      ContentActionSummarize,
			Description: "Create a summary of content",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content":   map[string]string{"type": "string"},
					"maxLength": map[string]string{"type": "number"},
				},
				"required": []string{"content"},
			},
		},
		{
			Action:      ContentActionExpand,
			Description: "Expand content with more detail",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content": map[string]string{"type": "string"},
					"focus":   map[string]string{"type": "string"},
				},
				"required": []string{"content"},
			},
		},
		{
			Action:      ContentActionEdit,
			Description: "Apply specific edits to content",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content":      map[string]string{"type": "string"},
					"instructions": map[string]string{"type": "string"},
				},
				"required": []string{"content", "instructions"},
			},
		},
		{
			Action:      ContentActionProofread,
			Description: "Check and correct grammar and spelling",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content": map[string]string{"type": "string"},
				},
				"required": []string{"content"},
			},
		},
		{
			Action:      ContentActionToneAdjust,
			Description: "Adjust the tone and style of content",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content": map[string]string{"type": "string"},
					"tone":    map[string]string{"type": "string"},
				},
				"required": []string{"content", "tone"},
			},
		},
		{
			Action:      ContentActionExtractKeywords,
			Description: "Extract keywords from content",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content": map[string]string{"type": "string"},
					"count":   map[string]string{"type": "number"},
				},
				"required": []string{"content"},
			},
		},
		{
			Action:      ContentActionGenerateOutline,
			Description: "Generate an outline for content",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"topic": map[string]string{"type": "string"},
					"depth": map[string]string{"type": "number"},
				},
				"required": []string{"topic"},
			},
		},
		{
			Action:      ContentActionGenerateVariations,
			Description: "Generate variations of content",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content": map[string]string{"type": "string"},
					"count":   map[string]string{"type": "number"},
				},
				"required": []string{"content"},
			},
		},
	}
}

// CanHandle returns true if this agent can handle the given action.
func (a *ContentAgent) CanHandle(action string) bool {
	switch action {
	case ContentActionGenerate, ContentActionRewrite, ContentActionSummarize,
		ContentActionExpand, ContentActionEdit, ContentActionProofread,
		ContentActionToneAdjust, ContentActionExtractKeywords,
		ContentActionGenerateOutline, ContentActionGenerateVariations:
		return true
	default:
		return false
	}
}

// Execute performs an action with the given input.
func (a *ContentAgent) Execute(ctx context.Context, action string, input *agents.ActionInput) (*agents.ActionOutput, error) {
	switch action {
	case ContentActionGenerate:
		return a.generate(ctx, input)
	case ContentActionRewrite:
		return a.rewrite(ctx, input)
	case ContentActionSummarize:
		return a.summarize(ctx, input)
	case ContentActionExpand:
		return a.expand(ctx, input)
	case ContentActionEdit:
		return a.edit(ctx, input)
	case ContentActionProofread:
		return a.proofread(ctx, input)
	case ContentActionToneAdjust:
		return a.toneAdjust(ctx, input)
	case ContentActionExtractKeywords:
		return a.extractKeywords(ctx, input)
	case ContentActionGenerateOutline:
		return a.generateOutline(ctx, input)
	case ContentActionGenerateVariations:
		return a.generateVariations(ctx, input)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (a *ContentAgent) generate(ctx context.Context, input *agents.ActionInput) (*agents.ActionOutput, error) {
	prompt, _ := input.Data["prompt"].(string)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	constraints := getConstraints(input.Data)

	systemPrompt := buildSystemPrompt(constraints)
	userPrompt := fmt.Sprintf("Generate content based on this prompt: %s", prompt)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: systemPrompt},
			{Role: llm.RoleUser, Content: userPrompt},
		},
		MaxTokens:   constraints.maxTokens(),
		Temperature: 0.7,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	return &agents.ActionOutput{
		Success: true,
		Data: map[string]any{
			"content": resp.Content,
		},
		Messages: []agents.Message{
			{Role: "assistant", Content: "Content generated successfully."},
		},
		NextActions: []agents.SuggestedAction{
			{Agent: agents.AgentTypeSEO, Action: "analyze", Reason: "Analyze the generated content for SEO"},
		},
	}, nil
}

func (a *ContentAgent) rewrite(ctx context.Context, input *agents.ActionInput) (*agents.ActionOutput, error) {
	content, _ := input.Data["content"].(string)
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	instructions, _ := input.Data["instructions"].(string)
	if instructions == "" {
		instructions = "Improve clarity, flow, and engagement while maintaining the original meaning."
	}

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a professional editor. Rewrite content to improve it while maintaining the original message and tone."},
			{Role: llm.RoleUser, Content: fmt.Sprintf("Rewrite the following content. Instructions: %s\n\nContent:\n%s", instructions, content)},
		},
		MaxTokens:   4096,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	return &agents.ActionOutput{
		Success: true,
		Data: map[string]any{
			"content":  resp.Content,
			"original": content,
		},
		Messages: []agents.Message{
			{Role: "assistant", Content: "Content rewritten successfully."},
		},
	}, nil
}

func (a *ContentAgent) summarize(ctx context.Context, input *agents.ActionInput) (*agents.ActionOutput, error) {
	content, _ := input.Data["content"].(string)
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	maxLength := 200
	if ml, ok := input.Data["maxLength"].(float64); ok {
		maxLength = int(ml)
	}

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a skilled summarizer. Create concise, accurate summaries that capture the key points."},
			{Role: llm.RoleUser, Content: fmt.Sprintf("Summarize the following content in approximately %d words:\n\n%s", maxLength, content)},
		},
		MaxTokens:   1024,
		Temperature: 0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	return &agents.ActionOutput{
		Success: true,
		Data: map[string]any{
			"summary":  resp.Content,
			"original": content,
		},
		Messages: []agents.Message{
			{Role: "assistant", Content: "Content summarized successfully."},
		},
	}, nil
}

func (a *ContentAgent) expand(ctx context.Context, input *agents.ActionInput) (*agents.ActionOutput, error) {
	content, _ := input.Data["content"].(string)
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	focus, _ := input.Data["focus"].(string)
	focusInstruction := ""
	if focus != "" {
		focusInstruction = fmt.Sprintf("Focus on expanding: %s", focus)
	}

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a skilled content writer. Expand content with more detail, examples, and depth while maintaining coherence."},
			{Role: llm.RoleUser, Content: fmt.Sprintf("Expand the following content with more detail and examples. %s\n\nContent:\n%s", focusInstruction, content)},
		},
		MaxTokens:   4096,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	return &agents.ActionOutput{
		Success: true,
		Data: map[string]any{
			"content":  resp.Content,
			"original": content,
		},
		Messages: []agents.Message{
			{Role: "assistant", Content: "Content expanded successfully."},
		},
	}, nil
}

func (a *ContentAgent) edit(ctx context.Context, input *agents.ActionInput) (*agents.ActionOutput, error) {
	content, _ := input.Data["content"].(string)
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	instructions, _ := input.Data["instructions"].(string)
	if instructions == "" {
		return nil, fmt.Errorf("instructions are required")
	}

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a professional editor. Apply the requested edits to the content precisely."},
			{Role: llm.RoleUser, Content: fmt.Sprintf("Apply these edits to the content: %s\n\nContent:\n%s", instructions, content)},
		},
		MaxTokens:   4096,
		Temperature: 0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	return &agents.ActionOutput{
		Success: true,
		Data: map[string]any{
			"content":  resp.Content,
			"original": content,
		},
		Messages: []agents.Message{
			{Role: "assistant", Content: "Edits applied successfully."},
		},
	}, nil
}

func (a *ContentAgent) proofread(ctx context.Context, input *agents.ActionInput) (*agents.ActionOutput, error) {
	content, _ := input.Data["content"].(string)
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: `You are a professional proofreader. Check the content for:
1. Grammar errors
2. Spelling mistakes
3. Punctuation issues
4. Awkward phrasing

Return a JSON object with:
- "corrected": the corrected content
- "issues": array of {type, original, corrected, explanation}`},
			{Role: llm.RoleUser, Content: fmt.Sprintf("Proofread this content:\n\n%s", content)},
		},
		MaxTokens:   4096,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	// Try to parse as JSON
	var result map[string]any
	respContent := strings.TrimSpace(resp.Content)
	if start := strings.Index(respContent, "{"); start != -1 {
		if end := strings.LastIndex(respContent, "}"); end != -1 {
			if err := json.Unmarshal([]byte(respContent[start:end+1]), &result); err == nil {
				return &agents.ActionOutput{
					Success: true,
					Data:    result,
					Messages: []agents.Message{
						{Role: "assistant", Content: "Proofreading completed."},
					},
				}, nil
			}
		}
	}

	// Fallback to raw content
	return &agents.ActionOutput{
		Success: true,
		Data: map[string]any{
			"corrected": resp.Content,
			"original":  content,
		},
		Messages: []agents.Message{
			{Role: "assistant", Content: "Proofreading completed."},
		},
	}, nil
}

func (a *ContentAgent) toneAdjust(ctx context.Context, input *agents.ActionInput) (*agents.ActionOutput, error) {
	content, _ := input.Data["content"].(string)
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	tone, _ := input.Data["tone"].(string)
	if tone == "" {
		return nil, fmt.Errorf("tone is required")
	}

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a skilled writer who can adapt content to different tones and styles while preserving the core message."},
			{Role: llm.RoleUser, Content: fmt.Sprintf("Adjust the following content to have a %s tone:\n\n%s", tone, content)},
		},
		MaxTokens:   4096,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	return &agents.ActionOutput{
		Success: true,
		Data: map[string]any{
			"content":  resp.Content,
			"tone":     tone,
			"original": content,
		},
		Messages: []agents.Message{
			{Role: "assistant", Content: fmt.Sprintf("Content adjusted to %s tone.", tone)},
		},
	}, nil
}

func (a *ContentAgent) extractKeywords(ctx context.Context, input *agents.ActionInput) (*agents.ActionOutput, error) {
	content, _ := input.Data["content"].(string)
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	count := 10
	if c, ok := input.Data["count"].(float64); ok {
		count = int(c)
	}

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: fmt.Sprintf("Extract the %d most relevant keywords from the content. Return as JSON array of objects with 'keyword' and 'relevance' (0-1) fields.", count)},
			{Role: llm.RoleUser, Content: content},
		},
		MaxTokens:   500,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	// Parse keywords
	var keywords []map[string]any
	respContent := strings.TrimSpace(resp.Content)
	if start := strings.Index(respContent, "["); start != -1 {
		if end := strings.LastIndex(respContent, "]"); end != -1 {
			json.Unmarshal([]byte(respContent[start:end+1]), &keywords)
		}
	}

	return &agents.ActionOutput{
		Success: true,
		Data: map[string]any{
			"keywords": keywords,
		},
		Messages: []agents.Message{
			{Role: "assistant", Content: fmt.Sprintf("Extracted %d keywords.", len(keywords))},
		},
	}, nil
}

func (a *ContentAgent) generateOutline(ctx context.Context, input *agents.ActionInput) (*agents.ActionOutput, error) {
	topic, _ := input.Data["topic"].(string)
	if topic == "" {
		return nil, fmt.Errorf("topic is required")
	}

	depth := 2
	if d, ok := input.Data["depth"].(float64); ok {
		depth = int(d)
	}

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: fmt.Sprintf("Create a detailed content outline with %d levels of depth. Return as JSON with nested 'sections' arrays containing 'title' and 'description' fields.", depth)},
			{Role: llm.RoleUser, Content: fmt.Sprintf("Create an outline for: %s", topic)},
		},
		MaxTokens:   2048,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	// Parse outline
	var outline map[string]any
	respContent := strings.TrimSpace(resp.Content)
	if start := strings.Index(respContent, "{"); start != -1 {
		if end := strings.LastIndex(respContent, "}"); end != -1 {
			json.Unmarshal([]byte(respContent[start:end+1]), &outline)
		}
	}

	if outline == nil {
		outline = map[string]any{"raw": resp.Content}
	}

	return &agents.ActionOutput{
		Success: true,
		Data: map[string]any{
			"outline": outline,
			"topic":   topic,
		},
		Messages: []agents.Message{
			{Role: "assistant", Content: "Outline generated successfully."},
		},
		NextActions: []agents.SuggestedAction{
			{Agent: agents.AgentTypeContent, Action: ContentActionGenerate, Reason: "Generate content from this outline"},
		},
	}, nil
}

func (a *ContentAgent) generateVariations(ctx context.Context, input *agents.ActionInput) (*agents.ActionOutput, error) {
	content, _ := input.Data["content"].(string)
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	count := 3
	if c, ok := input.Data["count"].(float64); ok {
		count = int(c)
	}

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: fmt.Sprintf("Generate %d variations of the content. Each variation should convey the same message but with different wording, structure, or style. Return as JSON array.", count)},
			{Role: llm.RoleUser, Content: content},
		},
		MaxTokens:   4096,
		Temperature: 0.8,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	// Parse variations
	var variations []string
	respContent := strings.TrimSpace(resp.Content)
	if start := strings.Index(respContent, "["); start != -1 {
		if end := strings.LastIndex(respContent, "]"); end != -1 {
			json.Unmarshal([]byte(respContent[start:end+1]), &variations)
		}
	}

	return &agents.ActionOutput{
		Success: true,
		Data: map[string]any{
			"variations": variations,
			"original":   content,
		},
		Messages: []agents.Message{
			{Role: "assistant", Content: fmt.Sprintf("Generated %d variations.", len(variations))},
		},
	}, nil
}

// Helper types and functions

type contentConstraints struct {
	MinLength int
	MaxLength int
	Tone      string
	Style     string
	Keywords  []string
	Avoid     []string
	Format    string
}

func getConstraints(data map[string]any) contentConstraints {
	c := contentConstraints{
		Format: "markdown",
	}

	if constraints, ok := data["constraints"].(map[string]any); ok {
		if v, ok := constraints["minLength"].(float64); ok {
			c.MinLength = int(v)
		}
		if v, ok := constraints["maxLength"].(float64); ok {
			c.MaxLength = int(v)
		}
		if v, ok := constraints["tone"].(string); ok {
			c.Tone = v
		}
		if v, ok := constraints["style"].(string); ok {
			c.Style = v
		}
		if v, ok := constraints["format"].(string); ok {
			c.Format = v
		}
		if v, ok := constraints["keywords"].([]any); ok {
			for _, k := range v {
				if s, ok := k.(string); ok {
					c.Keywords = append(c.Keywords, s)
				}
			}
		}
		if v, ok := constraints["avoid"].([]any); ok {
			for _, k := range v {
				if s, ok := k.(string); ok {
					c.Avoid = append(c.Avoid, s)
				}
			}
		}
	}

	return c
}

func (c contentConstraints) maxTokens() int {
	if c.MaxLength > 0 {
		// Rough estimate: 1 word ≈ 1.3 tokens
		return int(float64(c.MaxLength) * 1.3)
	}
	return 4096
}

func buildSystemPrompt(c contentConstraints) string {
	var parts []string
	parts = append(parts, "You are a professional content writer. Generate high-quality content that is engaging, informative, and well-structured.")

	if c.Tone != "" {
		parts = append(parts, fmt.Sprintf("Use a %s tone.", c.Tone))
	}
	if c.Style != "" {
		parts = append(parts, fmt.Sprintf("Write in a %s style.", c.Style))
	}
	if c.MinLength > 0 {
		parts = append(parts, fmt.Sprintf("The content should be at least %d words.", c.MinLength))
	}
	if c.MaxLength > 0 {
		parts = append(parts, fmt.Sprintf("The content should not exceed %d words.", c.MaxLength))
	}
	if len(c.Keywords) > 0 {
		parts = append(parts, fmt.Sprintf("Include these keywords naturally: %s", strings.Join(c.Keywords, ", ")))
	}
	if len(c.Avoid) > 0 {
		parts = append(parts, fmt.Sprintf("Avoid using: %s", strings.Join(c.Avoid, ", ")))
	}
	if c.Format != "" {
		parts = append(parts, fmt.Sprintf("Format the output as %s.", c.Format))
	}

	return strings.Join(parts, " ")
}

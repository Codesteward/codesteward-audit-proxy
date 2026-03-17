package parser

import (
	"testing"
)

func TestParseSAPAICore_HarmonizedToolCall(t *testing.T) {
	body := mustJSON(map[string]any{
		"model": "gpt-4o-sap",
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"content": nil,
					"tool_calls": []any{
						map[string]any{
							"id":   "call_1",
							"type": "function",
							"function": map[string]any{
								"name":      "Read",
								"arguments": `{"file_path":"main.go"}`,
							},
						},
					},
				},
			},
		},
	})

	result, err := ParseSAPAICore(body, false,
		"/v2/inference/deployments/d12345/chat/completions", "my-rg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Model != "d12345" {
		t.Errorf("model: got %q, want %q", result.Model, "d12345")
	}
	if result.ResourceGroup != "my-rg" {
		t.Errorf("resource_group: got %q, want %q", result.ResourceGroup, "my-rg")
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("tool calls: got %d, want 1", len(result.ToolCalls))
	}
	if result.ToolCalls[0].Name != "Read" {
		t.Errorf("tool name: got %q, want %q", result.ToolCalls[0].Name, "Read")
	}
}

func TestParseSAPAICore_MissingResourceGroup(t *testing.T) {
	body := mustJSON(map[string]any{
		"model": "gpt-4o-sap",
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"content": "Hello from SAP",
				},
			},
		},
	})

	result, err := ParseSAPAICore(body, false,
		"/v2/inference/deployments/d12345/chat/completions", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ResourceGroup != "" {
		t.Errorf("resource_group: got %q, want empty", result.ResourceGroup)
	}
	if result.Model != "d12345" {
		t.Errorf("model: got %q, want %q", result.Model, "d12345")
	}
	if len(result.ToolCalls) != 0 {
		t.Errorf("unexpected tool calls: %v", result.ToolCalls)
	}
}

func TestParseSAPAICore_NativeAPIMode(t *testing.T) {
	body := []byte(`{"some":"arbitrary","native":"response"}`)

	result, err := ParseSAPAICore(body, false,
		"/v2/inference/deployments/d99/invoke", "rg-native")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Native mode: no tool calls, no assistant text, no panic.
	if len(result.ToolCalls) != 0 {
		t.Errorf("expected no tool calls for native mode, got %v", result.ToolCalls)
	}
	if len(result.AssistantText) != 0 {
		t.Errorf("expected no assistant text for native mode, got %v", result.AssistantText)
	}
	// Deployment ID still extracted from path.
	if result.Model != "d99" {
		t.Errorf("model: got %q, want %q", result.Model, "d99")
	}
}

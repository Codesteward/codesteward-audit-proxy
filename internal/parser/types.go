package parser

// ToolCall holds the name and JSON-encoded input for a single tool invocation.
type ToolCall struct {
	Name  string
	Input string // JSON object as a string
}

// TokenUsage holds normalised token counts from an LLM response.
// Field names are provider-agnostic; both Anthropic and OpenAI formats
// map into this struct.
type TokenUsage struct {
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int // Anthropic cache_read_input_tokens
	CacheWriteTokens int // Anthropic cache_creation_input_tokens
}

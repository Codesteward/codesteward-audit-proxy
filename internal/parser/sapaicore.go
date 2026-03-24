package parser

import (
	"log/slog"
	"regexp"
	"strings"
)

// deploymentIDRe matches /v2/inference/deployments/{id}/chat/completions
var deploymentIDRe = regexp.MustCompile(`/deployments/([^/]+)/`)

// SAPAICoreResult holds normalized content from an SAP AI Core response.
type SAPAICoreResult struct {
	AssistantText []string
	ToolCalls     []ToolCall
	Model         string // deployment ID
	ResourceGroup string
	Usage         TokenUsage
}

// ParseSAPAICore parses an SAP AI Core Harmonized API response.
// reqPath is the original request URL path (used to extract deployment ID).
// resourceGroup is the value of the AI-Resource-Group request header.
func ParseSAPAICore(body []byte, isStream bool, reqPath, resourceGroup string) (SAPAICoreResult, error) {
	result := SAPAICoreResult{ResourceGroup: resourceGroup}

	// Extract deployment ID from path.
	if m := deploymentIDRe.FindStringSubmatch(reqPath); len(m) == 2 {
		result.Model = m[1]
	}

	// Native API Mode (/invoke suffix) — format may differ from OpenAI.
	// TODO: Implement Native API Mode response parsing.
	if strings.HasSuffix(strings.TrimRight(reqPath, "/"), "/invoke") {
		slog.Warn("sap-ai-core: Native API Mode response not parsed; storing raw only",
			"path", reqPath)
		return result, nil
	}

	// Harmonized API: delegate entirely to the OpenAI parser.
	oaiResult, err := ParseOpenAI(body, isStream)
	if err != nil {
		return result, err
	}
	result.AssistantText = oaiResult.AssistantText
	result.ToolCalls = oaiResult.ToolCalls
	result.Usage = oaiResult.Usage
	// Model from OpenAI response is the SAP model alias; prefer deployment ID
	// from path (already set above); only fall back if path had no match.
	if result.Model == "" {
		result.Model = oaiResult.Model
	}
	return result, nil
}

package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Upstream identifies a target LLM API backend.
type Upstream struct {
	// Name is a short identifier used in logs and audit records.
	Name string
	// URL is the base URL of the upstream API.
	URL *url.URL
}

const (
	defaultAnthropicURL = "https://api.anthropic.com"
	defaultOpenAIURL    = "https://api.openai.com"
	defaultGeminiURL    = "https://generativelanguage.googleapis.com"
)

func mustParseUpstream(name, rawURL string) *Upstream {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(fmt.Sprintf("router: failed to parse upstream URL %q: %v", rawURL, err))
	}
	return &Upstream{Name: name, URL: u}
}

func parseUpstreamOrDefault(name, override, defaultURL string) *Upstream {
	if override != "" {
		u, err := url.Parse(override)
		if err == nil {
			return &Upstream{Name: name, URL: u}
		}
	}
	return mustParseUpstream(name, defaultURL)
}

// Router holds per-instance routing config (upstream URL overrides, SAP AI Core).
type Router struct {
	anthropic   *Upstream
	openai      *Upstream
	gemini      *Upstream
	sapUpstream *Upstream // nil when SAP_AICORE_BASE_URL unset
	sapAuthHost string
}

// RouterConfig holds optional upstream URL overrides and SAP AI Core settings.
type RouterConfig struct {
	AnthropicUpstreamURL string
	OpenAIUpstreamURL    string
	GeminiUpstreamURL    string
	SAPAICoreBaseURL     string
	SAPAICoreAuthHost    string
}

// NewRouter builds a Router. Fields in cfg may be empty to use defaults.
func NewRouter(sapBaseURL, sapAuthHost string) *Router {
	return NewRouterWithConfig(RouterConfig{
		SAPAICoreBaseURL:  sapBaseURL,
		SAPAICoreAuthHost: sapAuthHost,
	})
}

// NewRouterWithConfig builds a Router with full upstream URL override support.
func NewRouterWithConfig(cfg RouterConfig) *Router {
	rt := &Router{
		anthropic:   parseUpstreamOrDefault("anthropic", cfg.AnthropicUpstreamURL, defaultAnthropicURL),
		openai:      parseUpstreamOrDefault("openai", cfg.OpenAIUpstreamURL, defaultOpenAIURL),
		gemini:      parseUpstreamOrDefault("gemini", cfg.GeminiUpstreamURL, defaultGeminiURL),
		sapAuthHost: cfg.SAPAICoreAuthHost,
	}
	if cfg.SAPAICoreBaseURL != "" {
		u, err := url.Parse(cfg.SAPAICoreBaseURL)
		if err == nil {
			rt.sapUpstream = &Upstream{Name: "sap-ai-core", URL: u}
		}
	}
	return rt
}

// DetectUpstream returns the appropriate upstream for the incoming request.
//
// Detection priority:
//  1. SAP AI Core — when configured, checked before other host-based rules.
//  2. Host header — exact match against known upstream hostnames.
//  3. Path prefix — /v1/messages → Anthropic, /v1/chat/ → OpenAI,
//     /v1beta/ → Gemini, /anthropic/ → Anthropic.
//  4. Presence of Anthropic-specific request headers.
//  5. Default: Anthropic (most common agent use-case is Claude Code).
func (rt *Router) DetectUpstream(r *http.Request) *Upstream {
	host := r.Host
	if host == "" {
		host = r.Header.Get("Host")
	}

	// Strip port if present.
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// SAP AI Core: check before other host-based checks.
	if rt.sapUpstream != nil && strings.Contains(host, rt.sapAuthHost) {
		return rt.sapUpstream
	}

	switch {
	case strings.EqualFold(host, "api.anthropic.com"):
		return rt.anthropic
	case strings.EqualFold(host, "api.openai.com"):
		return rt.openai
	case strings.EqualFold(host, "generativelanguage.googleapis.com"):
		return rt.gemini
	}

	path := r.URL.Path
	switch {
	case strings.HasPrefix(path, "/v1/messages"),
		strings.HasPrefix(path, "/anthropic/"):
		return rt.anthropic
	case strings.HasPrefix(path, "/v1/chat/"),
		strings.HasPrefix(path, "/openai/"):
		return rt.openai
	case strings.HasPrefix(path, "/v1beta/"),
		strings.HasPrefix(path, "/gemini/"):
		return rt.gemini
	}

	// Header-based hint: Anthropic requests always include anthropic-version.
	if r.Header.Get("anthropic-version") != "" {
		return rt.anthropic
	}

	return rt.anthropic
}

// RewriteRequest rewrites req.URL to point at the given upstream, preserving
// the original path and query string. It also updates the Host header so the
// upstream TLS certificate validation succeeds.
func RewriteRequest(req *http.Request, upstream *Upstream) {
	req.URL.Scheme = upstream.URL.Scheme
	req.URL.Host = upstream.URL.Host
	req.Host = upstream.URL.Host

	// Strip a path prefix used for local routing disambiguation
	// (e.g. /anthropic/v1/messages → /v1/messages).
	for _, prefix := range []string{"/anthropic/", "/openai/", "/gemini/", "/sap-aicore/"} {
		if strings.HasPrefix(req.URL.Path, prefix) {
			req.URL.Path = "/" + strings.TrimPrefix(req.URL.Path, prefix)
			break
		}
	}
}

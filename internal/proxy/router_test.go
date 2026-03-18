package proxy

import (
	"net/http"
	"net/url"
	"testing"
)

func newReq(host, path string) *http.Request {
	r := &http.Request{
		Host:   host,
		Header: make(http.Header),
		URL:    &url.URL{Path: path},
	}
	return r
}

// defaultRouter is a Router with SAP AI Core disabled, used across tests.
var defaultRouter = NewRouter("", "ml.hana.ondemand.com")

// --- DetectUpstream ---------------------------------------------------------

func TestDetectUpstream_HostAnthropic(t *testing.T) {
	r := newReq("api.anthropic.com", "/v1/messages")
	up := defaultRouter.DetectUpstream(r)
	if up.Name != "anthropic" {
		t.Errorf("got %q, want %q", up.Name, "anthropic")
	}
}

func TestDetectUpstream_HostOpenAI(t *testing.T) {
	r := newReq("api.openai.com", "/v1/chat/completions")
	up := defaultRouter.DetectUpstream(r)
	if up.Name != "openai" {
		t.Errorf("got %q, want %q", up.Name, "openai")
	}
}

func TestDetectUpstream_HostGemini(t *testing.T) {
	r := newReq("generativelanguage.googleapis.com", "/v1beta/models/gemini-pro:generateContent")
	up := defaultRouter.DetectUpstream(r)
	if up.Name != "gemini" {
		t.Errorf("got %q, want %q", up.Name, "gemini")
	}
}

func TestDetectUpstream_HostWithPort(t *testing.T) {
	// When agent sets ANTHROPIC_BASE_URL=http://localhost:8080 the Host header
	// is "localhost:8080", so routing falls through to path-based detection.
	r := newReq("localhost:8080", "/v1/messages")
	up := defaultRouter.DetectUpstream(r)
	if up.Name != "anthropic" {
		t.Errorf("got %q, want %q", up.Name, "anthropic")
	}
}

func TestDetectUpstream_PathMessages(t *testing.T) {
	r := newReq("localhost:8080", "/v1/messages")
	up := defaultRouter.DetectUpstream(r)
	if up.Name != "anthropic" {
		t.Errorf("got %q, want %q", up.Name, "anthropic")
	}
}

func TestDetectUpstream_PathChatCompletions(t *testing.T) {
	r := newReq("localhost:8080", "/v1/chat/completions")
	up := defaultRouter.DetectUpstream(r)
	if up.Name != "openai" {
		t.Errorf("got %q, want %q", up.Name, "openai")
	}
}

func TestDetectUpstream_PathV1Beta(t *testing.T) {
	r := newReq("localhost:8080", "/v1beta/models/gemini-pro:generateContent")
	up := defaultRouter.DetectUpstream(r)
	if up.Name != "gemini" {
		t.Errorf("got %q, want %q", up.Name, "gemini")
	}
}

func TestDetectUpstream_PathAnthropicPrefix(t *testing.T) {
	r := newReq("localhost:8080", "/anthropic/v1/messages")
	up := defaultRouter.DetectUpstream(r)
	if up.Name != "anthropic" {
		t.Errorf("got %q, want %q", up.Name, "anthropic")
	}
}

func TestDetectUpstream_AnthropicVersionHeader(t *testing.T) {
	r := newReq("localhost:8080", "/unknown/path")
	r.Header.Set("anthropic-version", "2023-06-01")
	up := defaultRouter.DetectUpstream(r)
	if up.Name != "anthropic" {
		t.Errorf("got %q, want %q", up.Name, "anthropic")
	}
}

func TestDetectUpstream_DefaultIsAnthropic(t *testing.T) {
	r := newReq("localhost:8080", "/")
	up := defaultRouter.DetectUpstream(r)
	if up.Name != "anthropic" {
		t.Errorf("default upstream: got %q, want %q", up.Name, "anthropic")
	}
}

// --- RewriteRequest ---------------------------------------------------------

func TestRewriteRequest_SetsSchemeAndHost(t *testing.T) {
	r := newReq("localhost:8080", "/v1/messages")
	r.URL = &url.URL{Path: "/v1/messages", RawQuery: "stream=true"}

	RewriteRequest(r, defaultRouter.anthropic)

	if r.URL.Scheme != "https" {
		t.Errorf("scheme: got %q, want %q", r.URL.Scheme, "https")
	}
	if r.URL.Host != "api.anthropic.com" {
		t.Errorf("host: got %q, want %q", r.URL.Host, "api.anthropic.com")
	}
	if r.Host != "api.anthropic.com" {
		t.Errorf("r.Host: got %q, want %q", r.Host, "api.anthropic.com")
	}
	// Query string must be preserved.
	if r.URL.RawQuery != "stream=true" {
		t.Errorf("query: got %q", r.URL.RawQuery)
	}
}

func TestRewriteRequest_StripsAnthropicPrefix(t *testing.T) {
	r := newReq("localhost:8080", "/anthropic/v1/messages")
	r.URL = &url.URL{Path: "/anthropic/v1/messages"}

	RewriteRequest(r, defaultRouter.anthropic)

	if r.URL.Path != "/v1/messages" {
		t.Errorf("path: got %q, want %q", r.URL.Path, "/v1/messages")
	}
}

func TestRewriteRequest_StripsOpenAIPrefix(t *testing.T) {
	r := newReq("localhost:8080", "/openai/v1/chat/completions")
	r.URL = &url.URL{Path: "/openai/v1/chat/completions"}

	RewriteRequest(r, defaultRouter.openai)

	if r.URL.Path != "/v1/chat/completions" {
		t.Errorf("path: got %q, want %q", r.URL.Path, "/v1/chat/completions")
	}
}

func TestRewriteRequest_PreservesNonPrefixedPath(t *testing.T) {
	r := newReq("localhost:8080", "/v1/messages")
	r.URL = &url.URL{Path: "/v1/messages"}

	RewriteRequest(r, defaultRouter.anthropic)

	if r.URL.Path != "/v1/messages" {
		t.Errorf("path should be unchanged: got %q", r.URL.Path)
	}
}

// --- Upstream URL overrides --------------------------------------------------

func TestNewRouterWithConfig_OverridesAnthropicURL(t *testing.T) {
	rt := NewRouterWithConfig(RouterConfig{
		AnthropicUpstreamURL: "https://litellm.internal/anthropic",
		SAPAICoreAuthHost:    "ml.hana.ondemand.com",
	})

	if rt.anthropic.URL.Host != "litellm.internal" {
		t.Errorf("host: got %q, want %q", rt.anthropic.URL.Host, "litellm.internal")
	}
	if rt.anthropic.URL.Path != "/anthropic" {
		t.Errorf("path: got %q, want %q", rt.anthropic.URL.Path, "/anthropic")
	}
}

func TestNewRouterWithConfig_DefaultsWhenOverrideEmpty(t *testing.T) {
	rt := NewRouterWithConfig(RouterConfig{SAPAICoreAuthHost: "ml.hana.ondemand.com"})

	if rt.anthropic.URL.Host != "api.anthropic.com" {
		t.Errorf("anthropic host: got %q, want %q", rt.anthropic.URL.Host, "api.anthropic.com")
	}
	if rt.openai.URL.Host != "api.openai.com" {
		t.Errorf("openai host: got %q, want %q", rt.openai.URL.Host, "api.openai.com")
	}
	if rt.gemini.URL.Host != "generativelanguage.googleapis.com" {
		t.Errorf("gemini host: got %q, want %q", rt.gemini.URL.Host, "generativelanguage.googleapis.com")
	}
}

func TestDetectUpstream_UsesOverriddenURL(t *testing.T) {
	rt := NewRouterWithConfig(RouterConfig{
		OpenAIUpstreamURL: "https://litellm.internal/openai",
		SAPAICoreAuthHost: "ml.hana.ondemand.com",
	})

	r := newReq("localhost:8080", "/v1/chat/completions")
	up := rt.DetectUpstream(r)

	if up.Name != "openai" {
		t.Errorf("name: got %q, want %q", up.Name, "openai")
	}
	if up.URL.Host != "litellm.internal" {
		t.Errorf("host: got %q, want %q", up.URL.Host, "litellm.internal")
	}
}

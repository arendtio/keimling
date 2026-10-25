package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDelegateProfiles(t *testing.T) {
	c := Config{DelegateFast: "small", DelegateBalanced: "regular", DelegateDeep: "large"}
	for _, test := range []struct{ p, m, e string }{{"fast", "small", "low"}, {"balanced", "regular", "medium"}, {"deep", "large", "high"}} {
		m, e, err := delegateProfile(c, test.p)
		if err != nil || m != test.m || e != test.e {
			t.Fatalf("%s: %s %s %v", test.p, m, e, err)
		}
	}
	if _, _, err := delegateProfile(c, "unknown"); err == nil {
		t.Fatal("unknown profile accepted")
	}
}
func TestDelegateIsolatedAndCounted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var p map[string]json.RawMessage
		if json.Unmarshal(body, &p) != nil {
			t.Error("invalid JSON")
		}
		if _, ok := p["tools"]; ok {
			t.Error("delegate must not have tools")
		}
		if string(p["tool_choice"]) != "\"none\"" {
			t.Errorf("tool choice %s", p["tool_choice"])
		}
		if strings.Contains(string(body), "previous chat") {
			t.Error("history leaked")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"Found two issues"}]}],"usage":{"input_tokens":120,"output_tokens":22}}`))
	}))
	defer server.Close()
	c := NewClient("test-key")
	c.HTTP = &http.Client{Transport: rewriteTransport{endpoint: server.URL, base: http.DefaultTransport}}
	answer, err := c.Delegate(context.Background(), "mock-model", "medium", DelegateArgs{Task: "Review config", Context: "unit content", Profile: "balanced"})
	if err != nil || answer != "Found two issues" {
		t.Fatalf("answer %q err %v", answer, err)
	}
	if c.InputTokens != 120 || c.OutputTokens != 22 {
		t.Fatalf("usage %d %d", c.InputTokens, c.OutputTokens)
	}
}

type rewriteTransport struct {
	endpoint string
	base     http.RoundTripper
}

func (r rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	copy := req.Clone(req.Context())
	u, _ := copy.URL.Parse(r.endpoint)
	copy.URL = u
	return r.base.RoundTrip(copy)
}

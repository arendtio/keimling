package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mark3labs/mcp-go/server/servertest"
)

func TestMCPOAuthRedirect(t *testing.T) {
	loop, err := mcpOAuthRedirect("http://127.0.0.1:7420", "")
	if err != nil || loop != "http://127.0.0.1:53682/oauth/callback" {
		t.Fatalf("loopback: %q %v", loop, err)
	}
	local, err := mcpOAuthRedirect("", "https://keimling.example")
	if err != nil || local != loop {
		t.Fatalf("empty origin: %q %v", local, err)
	}
	remote, err := mcpOAuthRedirect("https://keimling.example", "https://keimling.example")
	if err != nil || remote != "https://keimling.example/oauth/callback" {
		t.Fatalf("domain: %q %v", remote, err)
	}
	if _, err := mcpOAuthRedirect("http://files.example", ""); err == nil {
		t.Fatal("plain http domain should fail")
	}
	for _, u := range []string{loop, remote} {
		if strings.Contains(u, "keimling://") {
			t.Fatal(u)
		}
	}
}

func TestMCPApplicationType(t *testing.T) {
	if mcpApplicationType("http://127.0.0.1:53682/oauth/callback") != "native" {
		t.Fatal("native")
	}
	if mcpApplicationType("https://keimling.example/oauth/callback") != "web" {
		t.Fatal("web")
	}
}

func TestMCPPKCE(t *testing.T) {
	verifier, challenge, err := mcpPKCE()
	if err != nil || verifier == "" || challenge == "" {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if challenge != want {
		t.Fatalf("challenge %q", challenge)
	}
}

func TestMCPClientMetadataListsBothCallbacks(t *testing.T) {
	b, err := mcpClientMetadata("https://keimling.example")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		ClientID  string   `json:"client_id"`
		Redirects []string `json:"redirect_uris"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.ClientID != "https://keimling.example/oauth/client-metadata.json" {
		t.Fatal(doc.ClientID)
	}
	joined := strings.Join(doc.Redirects, " ")
	if !strings.Contains(joined, "http://127.0.0.1:53682/oauth/callback") || !strings.Contains(joined, "https://keimling.example/oauth/callback") {
		t.Fatal(joined)
	}
}

func TestMCPDiscoverAndRegister(t *testing.T) {
	var asURL string
	as := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			_ = json.NewEncoder(w).Encode(mcpAuthServer{
				AuthorizationEndpoint: asURL + "/auth",
				TokenEndpoint:         asURL + "/token",
				RegistrationEndpoint:  asURL + "/register",
				CodeChallengeMethods:  []string{"S256"},
			})
		case "/register":
			var body struct {
				ApplicationType string   `json:"application_type"`
				Redirects       []string `json:"redirect_uris"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.ApplicationType != "native" {
				t.Errorf("application_type %q", body.ApplicationType)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"client_id": "keimling-client"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer as.Close()
	asURL = as.URL
	resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/oauth-protected-resource" {
			_ = json.NewEncoder(w).Encode(mcpResourceMeta{Resource: "http://mcp.example/mcp", AuthorizationServers: []string{as.URL}})
			return
		}
		w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+as.URL+`/meta"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer resource.Close()
	// Point discovery at the resource root so the well-known path is on this server.
	res, got, err := mcpDiscover(context.Background(), resource.Client(), resource.URL)
	if err != nil {
		t.Fatal(err)
	}
	if res.Resource == "" || got.TokenEndpoint == "" {
		t.Fatalf("%+v %+v", res, got)
	}
	id, err := mcpRegisterClient(context.Background(), resource.Client(), got, "", mcpLoopbackRedirect(), "")
	if err != nil || id != "keimling-client" {
		t.Fatalf("%q %v", id, err)
	}
}

func TestMCPRefreshPersistsTokenNotHeader(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(mcpTokenSet{AccessToken: "new-access", RefreshToken: "new-refresh", ExpiresIn: 3600})
	}))
	defer tokenSrv.Close()
	cfg := MCPServerConfig{
		URL: "http://mcp.example/mcp",
		OAuth: &MCPOAuth{
			ClientID:     "c",
			TokenURL:     tokenSrv.URL,
			RefreshToken: "old-refresh",
			ExpiresAt:    time.Now().Add(-time.Hour),
		},
	}
	if err := UpsertUserMCPServer("demo", cfg); err != nil {
		t.Fatal(err)
	}
	got, err := prepareMCPConnection(context.Background(), "demo", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got.Headers["Authorization"] != "Bearer new-access" {
		t.Fatalf("dial header %v", got.Headers)
	}
	saved := LoadMCPServerConfigs("")["demo"]
	if saved.OAuth == nil || saved.OAuth.AccessToken != "new-access" {
		t.Fatalf("saved %#v", saved.OAuth)
	}
	if saved.Headers["Authorization"] != "" {
		t.Fatal("token was written into headers")
	}
	raw, _ := json.Marshal((&MCPManager{sessions: map[string]*mcpSession{
		"demo": {name: "demo", cfg: saved, status: "connected"},
	}}).Status())
	if strings.Contains(string(raw), "new-access") || strings.Contains(string(raw), "new-refresh") {
		t.Fatal(string(raw))
	}
}

func TestSignInUsesChallengeScopeNotStoredScope(t *testing.T) {
	var asURL, resourceURL string
	as := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/oauth-authorization-server" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(mcpAuthServer{
			AuthorizationEndpoint: asURL + "/auth",
			TokenEndpoint:         asURL + "/token",
			RegistrationEndpoint:  asURL + "/register",
			CodeChallengeMethods:  []string{"S256"},
		})
	}))
	defer as.Close()
	asURL = as.URL
	resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/oauth-protected-resource" {
			_ = json.NewEncoder(w).Encode(mcpResourceMeta{
				Resource:             resourceURL + "/mcp",
				AuthorizationServers: []string{as.URL},
				Scopes:               []string{"old:read", "old:write"},
			})
			return
		}
		w.Header().Set("WWW-Authenticate", `Bearer realm="mcp", error="invalid_scope", scope="fresh:read fresh:write", resource_metadata="`+resourceURL+`/.well-known/oauth-protected-resource"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer resource.Close()
	resourceURL = resource.URL
	authURL, err := startMCPSignIn(context.Background(), resource.Client(), "demo", resource.URL+"/mcp", "http://127.0.0.1:7420", "", &MCPOAuth{Scope: "stored:old", ClientID: "already"})
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	if got := u.Query().Get("scope"); got != "fresh:read fresh:write" {
		t.Fatalf("scope %q in %s", got, authURL)
	}
	if strings.Contains(u.Query().Get("scope"), "old:") || strings.Contains(u.Query().Get("scope"), "stored:") {
		t.Fatal(authURL)
	}
}

func TestAuthParamIgnoresInsufficientScope(t *testing.T) {
	h := `Bearer error="insufficient_scope", scope="files:read", resource_metadata="https://mcp.example/meta"`
	if got := authParam(h, "scope"); got != "files:read" {
		t.Fatalf("scope %q", got)
	}
	if got := quotedParam(h, "resource_metadata"); got != "https://mcp.example/meta" {
		t.Fatalf("metadata %q", got)
	}
}

func TestOAuthRefreshTokenStaysSignedIn(t *testing.T) {
	cfg := MCPServerConfig{OAuth: &MCPOAuth{
		AccessToken:  "old",
		RefreshToken: "keep",
		TokenURL:     "https://login.example/token",
		ExpiresAt:    time.Now().Add(-time.Hour),
	}}
	if got := cfg.oauthState(time.Now()); got != "signed_in" {
		t.Fatalf("oauth %q", got)
	}
}

// mcpAuthProxy returns 401 for tools/call until the bearer is the renewed one.
func mcpAuthProxy(upstream string, accept string, rejected *atomic.Int32) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method == http.MethodPost && bytes.Contains(body, []byte(`"tools/call"`)) && r.Header.Get("Authorization") != accept {
			rejected.Add(1)
			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		req, err := http.NewRequestWithContext(r.Context(), r.Method, upstream+r.URL.RequestURI(), bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		req.Header = r.Header.Clone()
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for k, vals := range resp.Header {
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	})
}

func TestExpiredAccessTokenRefreshesAndRetries(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	var refreshes atomic.Int32
	var gotScope, gotRefresh string
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		refreshes.Add(1)
		gotScope = r.Form.Get("scope")
		gotRefresh = r.Form.Get("refresh_token")
		_ = json.NewEncoder(w).Encode(mcpTokenSet{AccessToken: "fresh", RefreshToken: "rotated", ExpiresIn: 3600})
	}))
	defer tokenSrv.Close()

	mcpSrv := server.NewMCPServer("auth", "1.0.0", server.WithToolCapabilities(true))
	mcpSrv.AddTool(mcp.NewTool("ping"), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("pong"), nil
	})
	upstream := servertest.NewTestStreamableHTTPServer(mcpSrv)
	defer upstream.Close()
	var rejected atomic.Int32
	proxy := httptest.NewServer(mcpAuthProxy(upstream.URL, "Bearer fresh", &rejected))
	defer proxy.Close()

	cfg := MCPServerConfig{
		URL: proxy.URL,
		OAuth: &MCPOAuth{
			ClientID:     "entra-client",
			TokenURL:     tokenSrv.URL,
			RefreshToken: "old-refresh",
			AccessToken:  "stale",
			Scope:        "User.Read",
			ExpiresAt:    time.Now().Add(time.Hour),
		},
	}
	if err := UpsertUserMCPServer("demo", cfg); err != nil {
		t.Fatal(err)
	}
	m := NewMCPManager()
	defer m.Close()
	m.ConnectAll(context.Background(), map[string]MCPServerConfig{"demo": cfg})
	if st := m.Status(); len(st) != 1 || st[0].Status != "connected" {
		t.Fatalf("connect %+v", st)
	}
	name := mcpExposedName("demo", "ping")
	var wg sync.WaitGroup
	got := make([]string, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			got[i] = m.Call(context.Background(), name, "")
		}(i)
	}
	wg.Wait()
	for i, g := range got {
		if g != "pong" {
			t.Fatalf("call %d: %q", i, g)
		}
	}
	if refreshes.Load() != 1 {
		t.Fatalf("refreshes %d", refreshes.Load())
	}
	if gotScope != "User.Read" || gotRefresh != "old-refresh" {
		t.Fatalf("scope %q refresh %q", gotScope, gotRefresh)
	}
	if rejected.Load() < 1 {
		t.Fatal("expected the stale token to be rejected once")
	}
	saved := LoadMCPServerConfigs("")["demo"]
	if saved.OAuth == nil || saved.OAuth.AccessToken != "fresh" || saved.OAuth.RefreshToken != "rotated" {
		t.Fatalf("saved %#v", saved.OAuth)
	}
	if saved.Headers["Authorization"] != "" {
		t.Fatalf("header stored %v", saved.Headers)
	}
}

func TestRejectedRefreshTokenAsksForSignInOnce(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	var refreshes atomic.Int32
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshes.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer tokenSrv.Close()

	mcpSrv := server.NewMCPServer("auth", "1.0.0", server.WithToolCapabilities(true))
	mcpSrv.AddTool(mcp.NewTool("ping"), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("pong"), nil
	})
	upstream := servertest.NewTestStreamableHTTPServer(mcpSrv)
	defer upstream.Close()
	var rejected atomic.Int32
	proxy := httptest.NewServer(mcpAuthProxy(upstream.URL, "Bearer fresh", &rejected))
	defer proxy.Close()

	cfg := MCPServerConfig{
		URL: proxy.URL,
		OAuth: &MCPOAuth{
			ClientID:     "entra-client",
			TokenURL:     tokenSrv.URL,
			RefreshToken: "dead-refresh",
			AccessToken:  "stale",
			Scope:        "User.Read",
			ExpiresAt:    time.Now().Add(time.Hour),
		},
	}
	if err := UpsertUserMCPServer("demo", cfg); err != nil {
		t.Fatal(err)
	}
	m := NewMCPManager()
	defer m.Close()
	m.ConnectAll(context.Background(), map[string]MCPServerConfig{"demo": cfg})
	got := m.Call(context.Background(), mcpExposedName("demo", "ping"), "")
	if !strings.Contains(got, "sign-in expired") || strings.Contains(got, "pong") {
		t.Fatalf("call %q", got)
	}
	if refreshes.Load() != 1 || rejected.Load() != 1 {
		t.Fatalf("refreshes %d rejected %d", refreshes.Load(), rejected.Load())
	}
	st := m.Status()
	if len(st) != 1 || !strings.Contains(st[0].Error, "sign-in expired") {
		t.Fatalf("status %+v", st)
	}
}

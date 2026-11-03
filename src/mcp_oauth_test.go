package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

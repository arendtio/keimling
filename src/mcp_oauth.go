package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	mcpOAuthLoopbackHost = "127.0.0.1:53682"
	mcpOAuthCallbackPath = "/oauth/callback"
	mcpOAuthMetaPath     = "/oauth/client-metadata.json"
)

func mcpLoopbackRedirect() string {
	return "http://" + mcpOAuthLoopbackHost + mcpOAuthCallbackPath
}

func (c MCPServerConfig) authKind() string {
	if c.OAuth != nil && (c.OAuth.AccessToken != "" || c.OAuth.RefreshToken != "") {
		return "oauth"
	}
	if len(c.Headers) > 0 {
		if _, ok := c.Headers["Authorization"]; ok && len(c.Headers) == 1 {
			return "bearer"
		}
		return "headers"
	}
	if len(c.Env) > 0 {
		return "env"
	}
	return ""
}

func (c MCPServerConfig) oauthState(now time.Time) string {
	if c.OAuth == nil {
		return ""
	}
	valid := c.OAuth.AccessToken != "" && (c.OAuth.ExpiresAt.IsZero() || now.Before(c.OAuth.ExpiresAt))
	if valid {
		return "signed_in"
	}
	if c.OAuth.RefreshToken != "" || c.OAuth.AccessToken != "" {
		return "expired"
	}
	return ""
}

// mcpOAuthRedirect picks the callback the browser can deliver to this process.
// A loopback page uses 127.0.0.1. A page on the server's own https address uses that address.
func mcpOAuthRedirect(pageOrigin, publicBase string) (string, error) {
	origin := strings.TrimSpace(pageOrigin)
	if origin == "" || origin == "null" || isLoopbackOrigin(origin) {
		return mcpLoopbackRedirect(), nil
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("sign-in needs the page address")
	}
	if u.Scheme == "https" {
		return strings.TrimRight(u.Scheme+"://"+u.Host, "/") + mcpOAuthCallbackPath, nil
	}
	pub, err := url.Parse(strings.TrimSpace(publicBase))
	if err == nil && pub.Scheme == "https" && strings.EqualFold(pub.Host, u.Host) {
		return strings.TrimRight(pub.Scheme+"://"+pub.Host, "/") + mcpOAuthCallbackPath, nil
	}
	return "", fmt.Errorf("sign-in from this address needs https on the server's own domain")
}

func isLoopbackOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

type mcpAuthServer struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	RegistrationEndpoint  string   `json:"registration_endpoint"`
	CodeChallengeMethods  []string `json:"code_challenge_methods_supported"`
	CIMDSupported         bool     `json:"client_id_metadata_document_supported"`
}

type mcpResourceMeta struct {
	Resource             string   `json:"resource"`
	AuthorizationServers []string `json:"authorization_servers"`
	Scopes               []string `json:"scopes_supported"`
}

func mcpPKCE() (verifier, challenge string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func mcpClientMetadata(publicBase string) ([]byte, error) {
	base := strings.TrimRight(strings.TrimSpace(publicBase), "/")
	id := base + mcpOAuthMetaPath
	redirects := []string{mcpLoopbackRedirect(), "http://localhost:53682" + mcpOAuthCallbackPath}
	if strings.HasPrefix(base, "https://") {
		redirects = append(redirects, base+mcpOAuthCallbackPath)
	}
	doc := map[string]any{
		"client_id":                  id,
		"client_name":                "Keimling",
		"redirect_uris":              redirects,
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
	}
	return json.Marshal(doc)
}

func mcpApplicationType(redirect string) string {
	if isLoopbackOrigin(redirect) {
		return "native"
	}
	return "web"
}

type mcpOAuthFlow struct {
	Name        string
	RedirectURI string
	Verifier    string
	TokenURL    string
	ClientID    string
	Resource    string
	Scope       string
	Deadline    time.Time
}

var mcpOAuthFlows sync.Map

func mcpRememberFlow(state string, flow mcpOAuthFlow) {
	flow.Deadline = time.Now().Add(10 * time.Minute)
	mcpOAuthFlows.Store(state, flow)
}

func mcpTakeFlow(state string) (mcpOAuthFlow, bool) {
	v, ok := mcpOAuthFlows.LoadAndDelete(state)
	if !ok {
		return mcpOAuthFlow{}, false
	}
	flow := v.(mcpOAuthFlow)
	if time.Now().After(flow.Deadline) {
		return mcpOAuthFlow{}, false
	}
	return flow, true
}

func mcpDiscover(ctx context.Context, client *http.Client, resourceURL string) (mcpResourceMeta, mcpAuthServer, error) {
	var none mcpResourceMeta
	var as mcpAuthServer
	res, err := mcpFetchResourceMeta(ctx, client, resourceURL)
	if err != nil {
		return none, as, err
	}
	if len(res.AuthorizationServers) == 0 {
		return none, as, fmt.Errorf("the server did not name an authorization server")
	}
	as, err = mcpFetchAuthServer(ctx, client, res.AuthorizationServers[0])
	if err != nil {
		return res, as, err
	}
	if !containsString(as.CodeChallengeMethods, "S256") {
		return res, as, fmt.Errorf("the authorization server does not advertise PKCE S256")
	}
	if as.AuthorizationEndpoint == "" || as.TokenEndpoint == "" {
		return res, as, fmt.Errorf("the authorization server is missing an endpoint")
	}
	return res, as, nil
}

func mcpFetchResourceMeta(ctx context.Context, client *http.Client, resourceURL string) (mcpResourceMeta, error) {
	var meta mcpResourceMeta
	metaURL := resourceMetadataURL("", resourceURL)
	meta, err := mcpGetJSON[mcpResourceMeta](ctx, client, metaURL)
	if err == nil && len(meta.AuthorizationServers) > 0 {
		return meta, nil
	}
	req, err2 := http.NewRequestWithContext(ctx, http.MethodGet, resourceURL, nil)
	if err2 != nil {
		if err != nil {
			return meta, err
		}
		return meta, err2
	}
	resp, err2 := client.Do(req)
	if err2 != nil {
		if err != nil {
			return meta, err
		}
		return meta, err2
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	found := resourceMetadataURL(resp.Header.Get("WWW-Authenticate"), resourceURL)
	if found == metaURL && err != nil {
		return meta, err
	}
	return mcpGetJSON[mcpResourceMeta](ctx, client, found)
}

func resourceMetadataURL(wwwAuth, resourceURL string) string {
	if v := quotedParam(wwwAuth, "resource_metadata"); v != "" {
		return v
	}
	u, err := url.Parse(resourceURL)
	if err != nil {
		return resourceURL
	}
	path := strings.Trim(u.Path, "/")
	u.Path = "/.well-known/oauth-protected-resource"
	u.RawQuery = ""
	u.Fragment = ""
	if path != "" {
		u.Path += "/" + path
	}
	return u.String()
}

func quotedParam(header, name string) string {
	key := strings.ToLower(name) + `="`
	i := strings.Index(strings.ToLower(header), key)
	if i < 0 {
		return ""
	}
	rest := header[i+len(key):]
	j := strings.Index(rest, `"`)
	if j < 0 {
		return ""
	}
	return rest[:j]
}

func mcpFetchAuthServer(ctx context.Context, client *http.Client, issuer string) (mcpAuthServer, error) {
	issuer = strings.TrimRight(issuer, "/")
	candidates := []string{
		issuer + "/.well-known/oauth-authorization-server",
		issuer + "/.well-known/openid-configuration",
	}
	var last error
	for _, u := range candidates {
		as, err := mcpGetJSON[mcpAuthServer](ctx, client, u)
		if err == nil && as.AuthorizationEndpoint != "" {
			return as, nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("no authorization server metadata")
	}
	return mcpAuthServer{}, last
}

func mcpGetJSON[T any](ctx context.Context, client *http.Client, raw string) (T, error) {
	var out T
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return out, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return out, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("metadata %s returned %d", raw, resp.StatusCode)
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return out, err
	}
	return out, nil
}

func mcpRegisterClient(ctx context.Context, client *http.Client, as mcpAuthServer, publicBase, redirect string, existingClient string) (string, error) {
	if strings.TrimSpace(existingClient) != "" {
		return existingClient, nil
	}
	pub := strings.TrimRight(strings.TrimSpace(publicBase), "/")
	if as.CIMDSupported && strings.HasPrefix(pub, "https://") {
		return pub + mcpOAuthMetaPath, nil
	}
	if as.RegistrationEndpoint == "" {
		return "", fmt.Errorf("this server has no way to register Keimling. Allow this callback URL: %s", redirect)
	}
	body := map[string]any{
		"client_name": "Keimling",
		"redirect_uris": []string{
			redirect,
		},
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
		"application_type":           mcpApplicationType(redirect),
	}
	if mcpApplicationType(redirect) == "native" {
		body["redirect_uris"] = []string{redirect, "http://localhost:53682" + mcpOAuthCallbackPath}
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, as.RegistrationEndpoint, strings.NewReader(string(raw)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("registration failed (%d). Allow this callback URL: %s", resp.StatusCode, redirect)
	}
	var doc struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal(b, &doc); err != nil || doc.ClientID == "" {
		return "", fmt.Errorf("registration did not return a client id. Allow this callback URL: %s", redirect)
	}
	return doc.ClientID, nil
}

func mcpAuthorizeURL(as mcpAuthServer, clientID, redirect, challenge, state, resource, scope string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirect)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	if resource != "" {
		q.Set("resource", resource)
	}
	if scope != "" {
		q.Set("scope", scope)
	}
	return as.AuthorizationEndpoint + "?" + q.Encode()
}

type mcpTokenSet struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

func mcpExchangeCode(ctx context.Context, client *http.Client, tokenURL, clientID, redirect, verifier, code, resource string) (mcpTokenSet, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirect)
	form.Set("client_id", clientID)
	form.Set("code_verifier", verifier)
	if resource != "" {
		form.Set("resource", resource)
	}
	return mcpTokenPOST(ctx, client, tokenURL, form)
}

func mcpRefreshAccess(ctx context.Context, client *http.Client, tokenURL, clientID, refresh, resource string) (mcpTokenSet, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refresh)
	form.Set("client_id", clientID)
	if resource != "" {
		form.Set("resource", resource)
	}
	return mcpTokenPOST(ctx, client, tokenURL, form)
}

func mcpTokenPOST(ctx context.Context, client *http.Client, tokenURL string, form url.Values) (mcpTokenSet, error) {
	var set mcpTokenSet
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return set, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return set, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	_ = json.Unmarshal(b, &set)
	if set.Error != "" {
		return set, fmt.Errorf("%s", set.Error)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || set.AccessToken == "" {
		return set, fmt.Errorf("token request failed")
	}
	return set, nil
}

func applyTokenSet(oauth *MCPOAuth, set mcpTokenSet, now time.Time) {
	oauth.AccessToken = set.AccessToken
	if set.RefreshToken != "" {
		oauth.RefreshToken = set.RefreshToken
	}
	if set.ExpiresIn > 0 {
		oauth.ExpiresAt = now.Add(time.Duration(set.ExpiresIn) * time.Second)
	}
}

func dialMCPConfig(cfg MCPServerConfig) MCPServerConfig {
	if cfg.OAuth == nil || cfg.OAuth.AccessToken == "" {
		return cfg
	}
	cfg.Headers = cloneHeaderMap(cfg.Headers)
	cfg.setBearer(cfg.OAuth.AccessToken)
	return cfg
}

func cloneHeaderMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func prepareMCPConnection(ctx context.Context, name string, cfg MCPServerConfig) (MCPServerConfig, error) {
	if cfg.OAuth == nil || (cfg.OAuth.AccessToken == "" && cfg.OAuth.RefreshToken == "") {
		return cfg, nil
	}
	now := time.Now()
	need := cfg.OAuth.AccessToken == "" || (!cfg.OAuth.ExpiresAt.IsZero() && !now.Before(cfg.OAuth.ExpiresAt.Add(-time.Minute)))
	if need && cfg.OAuth.RefreshToken != "" && cfg.OAuth.TokenURL != "" {
		set, err := mcpRefreshAccess(ctx, http.DefaultClient, cfg.OAuth.TokenURL, cfg.OAuth.ClientID, cfg.OAuth.RefreshToken, cfg.OAuth.Resource)
		if err != nil {
			return cfg, fmt.Errorf("sign-in expired: %w", err)
		}
		applyTokenSet(cfg.OAuth, set, now)
		if err := UpsertUserMCPServer(name, cfg); err != nil {
			return cfg, err
		}
	}
	return dialMCPConfig(cfg), nil
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if strings.EqualFold(s, want) {
			return true
		}
	}
	return false
}

var (
	mcpLoopbackOnce sync.Once
	mcpLoopbackErr  error
)

func startMCPSignIn(ctx context.Context, client *http.Client, name, resourceURL, pageOrigin, publicBase string, existing *MCPOAuth) (string, error) {
	redirect, err := mcpOAuthRedirect(pageOrigin, publicBase)
	if err != nil {
		return "", err
	}
	res, as, err := mcpDiscover(ctx, client, resourceURL)
	if err != nil {
		return "", err
	}
	prev := ""
	if existing != nil {
		prev = existing.ClientID
	}
	clientID, err := mcpRegisterClient(ctx, client, as, publicBase, redirect, prev)
	if err != nil {
		return "", err
	}
	verifier, challenge, err := mcpPKCE()
	if err != nil {
		return "", err
	}
	stateBuf := make([]byte, 32)
	if _, err := rand.Read(stateBuf); err != nil {
		return "", err
	}
	state := base64.RawURLEncoding.EncodeToString(stateBuf)
	scope := strings.Join(res.Scopes, " ")
	resource := res.Resource
	if resource == "" {
		resource = resourceURL
	}
	mcpRememberFlow(state, mcpOAuthFlow{
		Name:        name,
		RedirectURI: redirect,
		Verifier:    verifier,
		TokenURL:    as.TokenEndpoint,
		ClientID:    clientID,
		Resource:    resource,
		Scope:       scope,
	})
	return mcpAuthorizeURL(as, clientID, redirect, challenge, state, resource, scope), nil
}

func finishMCPSignIn(ctx context.Context, client *http.Client, state, code, configPath string) (string, error) {
	flow, ok := mcpTakeFlow(state)
	if !ok {
		return "", fmt.Errorf("sign-in expired")
	}
	set, err := mcpExchangeCode(ctx, client, flow.TokenURL, flow.ClientID, flow.RedirectURI, flow.Verifier, code, flow.Resource)
	if err != nil {
		return flow.Name, err
	}
	servers := LoadMCPServerConfigs(configPath)
	base := servers[flow.Name]
	if strings.TrimSpace(base.URL) == "" {
		return flow.Name, fmt.Errorf("server %q was not saved", flow.Name)
	}
	if base.OAuth == nil {
		base.OAuth = &MCPOAuth{}
	}
	base.OAuth.ClientID = flow.ClientID
	base.OAuth.TokenURL = flow.TokenURL
	base.OAuth.Resource = flow.Resource
	base.OAuth.Scope = flow.Scope
	applyTokenSet(base.OAuth, set, time.Now())
	if err := UpsertUserMCPServer(flow.Name, base); err != nil {
		return flow.Name, err
	}
	return flow.Name, nil
}

func ensureMCPLoopback(handler http.Handler) error {
	mcpLoopbackOnce.Do(func() {
		ln, err := net.Listen("tcp", mcpOAuthLoopbackHost)
		if err != nil {
			mcpLoopbackErr = fmt.Errorf("oauth callback port 53682 is in use")
			return
		}
		srv := &http.Server{Handler: handler}
		go func() { _ = srv.Serve(ln) }()
	})
	return mcpLoopbackErr
}

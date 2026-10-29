package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Credential is a stored secret metadata + value (value only in memory / vault plaintext).
type Credential struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Type         string   `json:"type"` // api_key | password | token | ssh_key
	Service      string   `json:"service,omitempty"`
	AllowedTools []string `json:"allowed_tools"`
	InjectArg    string   `json:"inject_arg,omitempty"` // MCP argument name receiving the secret
	InjectEnv    string   `json:"inject_env,omitempty"` // exec: env var name (vault-defined)
	Secret       string   `json:"secret,omitempty"`
}

// CredentialBroker holds the process-lifetime unlocked vault (if any).
type CredentialBroker struct {
	mu       sync.RWMutex
	path     string
	unlocked bool
	key      []byte
	salt     []byte
	creds    []Credential
}

func NewCredentialBroker(configPath string) *CredentialBroker {
	return &CredentialBroker{path: credentialsVaultPath(configPath), creds: []Credential{}}
}

func (b *CredentialBroker) Path() string {
	if b == nil {
		return ""
	}
	return b.path
}

func (b *CredentialBroker) Exists() bool {
	if b == nil || b.path == "" {
		return false
	}
	_, err := os.Stat(b.path)
	return err == nil
}

func (b *CredentialBroker) Unlocked() bool {
	if b == nil {
		return false
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.unlocked
}

func (b *CredentialBroker) Status() map[string]any {
	if b == nil {
		return map[string]any{"exists": false, "unlocked": false, "count": 0}
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	exists := false
	if b.path != "" {
		_, err := os.Stat(b.path)
		exists = err == nil
	}
	return map[string]any{
		"exists":   exists,
		"unlocked": b.unlocked,
		"count":    len(b.creds),
		"path":     vaultFileName,
	}
}

func (b *CredentialBroker) Unlock(password string) error {
	if b == nil {
		return fmt.Errorf("no credential broker")
	}
	password = strings.TrimSpace(password)
	if password == "" {
		return fmt.Errorf("password required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	env, err := readVaultEnvelope(b.path)
	if err != nil {
		return err
	}
	if len(env.Salt) == 0 {
		return fmt.Errorf("corrupt vault: missing salt")
	}
	key := deriveVaultKey(password, env.Salt)
	payload, err := decryptVaultPayload(key, env)
	if err != nil {
		return err
	}
	b.key = key
	b.salt = append([]byte(nil), env.Salt...)
	b.creds = payload.Credentials
	b.unlocked = true
	return nil
}

func (b *CredentialBroker) Create(password string) error {
	if b == nil {
		return fmt.Errorf("no credential broker")
	}
	password = strings.TrimSpace(password)
	if password == "" {
		return fmt.Errorf("password required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, err := os.Stat(b.path); err == nil {
		return fmt.Errorf("vault already exists; unlock instead")
	}
	salt, err := newVaultSalt()
	if err != nil {
		return err
	}
	key := deriveVaultKey(password, salt)
	env, err := encryptVaultPayload(key, vaultPayload{Credentials: []Credential{}})
	if err != nil {
		return err
	}
	env.Salt = salt
	if err := writeVaultEnvelope(b.path, env); err != nil {
		return err
	}
	b.key = key
	b.salt = salt
	b.creds = []Credential{}
	b.unlocked = true
	return nil
}

func (b *CredentialBroker) persistLocked() error {
	if !b.unlocked || b.key == nil {
		return fmt.Errorf("vault locked")
	}
	env, err := encryptVaultPayload(b.key, vaultPayload{Credentials: b.creds})
	if err != nil {
		return err
	}
	env.Salt = b.salt
	return writeVaultEnvelope(b.path, env)
}

func (b *CredentialBroker) ListPublic() []map[string]any {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]map[string]any, 0, len(b.creds))
	for _, c := range b.creds {
		out = append(out, c.public())
	}
	return out
}

func (c Credential) public() map[string]any {
	tools := c.AllowedTools
	if tools == nil {
		tools = []string{}
	}
	return map[string]any{
		"id":            c.ID,
		"name":          c.Name,
		"description":   c.Description,
		"type":          c.Type,
		"service":       c.Service,
		"allowed_tools": tools,
		"inject_arg":    c.InjectArg,
		"inject_env":    c.InjectEnv,
		"secret_set":    strings.TrimSpace(c.Secret) != "",
	}
}

func (b *CredentialBroker) Upsert(in Credential, secret string, replaceSecret bool) (Credential, error) {
	if b == nil {
		return Credential{}, fmt.Errorf("no credential broker")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.unlocked {
		return Credential{}, fmt.Errorf("vault locked")
	}
	in.ID = strings.TrimSpace(in.ID)
	in.Name = strings.TrimSpace(in.Name)
	in.Type = strings.ToLower(strings.TrimSpace(in.Type))
	in.Service = strings.TrimSpace(in.Service)
	in.Description = strings.TrimSpace(in.Description)
	in.InjectArg = strings.TrimSpace(in.InjectArg)
	in.InjectEnv = strings.TrimSpace(in.InjectEnv)
	if in.Name == "" {
		return Credential{}, fmt.Errorf("name required")
	}
	switch in.Type {
	case "api_key", "password", "token", "ssh_key":
	case "":
		in.Type = "password"
	default:
		return Credential{}, fmt.Errorf("type must be api_key, password, token, or ssh_key")
	}
	if in.Type == "ssh_key" {
		in.InjectArg = ""
		in.InjectEnv = ""
	} else {
		if in.InjectArg == "" {
			in.InjectArg = defaultInjectArg(in.Type)
		}
		if err := validateInjectArg(in.InjectArg); err != nil {
			return Credential{}, err
		}
		if in.InjectEnv != "" {
			if err := validateInjectEnv(in.InjectEnv); err != nil {
				return Credential{}, err
			}
		}
	}
	in.AllowedTools = normalizeToolList(in.AllowedTools)
	if in.ID == "" {
		in.ID = fmt.Sprintf("cred-%s", time.Now().UTC().Format("20060102-150405.000000000"))
	}
	idx := -1
	for i := range b.creds {
		if b.creds[i].ID == in.ID {
			idx = i
			break
		}
	}
	if idx >= 0 {
		prev := b.creds[idx]
		if replaceSecret {
			if strings.TrimSpace(secret) == "" {
				return Credential{}, fmt.Errorf("secret required")
			}
			in.Secret = secret
		} else {
			in.Secret = prev.Secret
		}
		b.creds[idx] = in
	} else {
		if strings.TrimSpace(secret) == "" {
			return Credential{}, fmt.Errorf("secret required")
		}
		in.Secret = secret
		b.creds = append(b.creds, in)
	}
	if err := b.persistLocked(); err != nil {
		return Credential{}, err
	}
	out := in
	out.Secret = ""
	return out, nil
}

func (b *CredentialBroker) Delete(id string) error {
	if b == nil {
		return fmt.Errorf("no credential broker")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.unlocked {
		return fmt.Errorf("vault locked")
	}
	id = strings.TrimSpace(id)
	next := b.creds[:0]
	found := false
	for _, c := range b.creds {
		if c.ID == id {
			found = true
			continue
		}
		next = append(next, c)
	}
	if !found {
		return fmt.Errorf("credential not found")
	}
	b.creds = next
	return b.persistLocked()
}

// ListForTool returns public metadata for credentials allowed for tool.
func (b *CredentialBroker) ListForTool(tool string) ([]map[string]any, error) {
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return nil, fmt.Errorf("tool required")
	}
	if b == nil || !b.Unlocked() {
		return nil, fmt.Errorf("vault locked or unavailable")
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]map[string]any, 0)
	for _, c := range b.creds {
		if credentialAllowsTool(c, tool) {
			item := c.public()
			delete(item, "secret_set")
			delete(item, "allowed_tools")
			// Keep inject_env / type / inject_arg so the agent can form {{keimling:id}} placeholders.
			out = append(out, item)
		}
	}
	return out, nil
}

// ResolveSecret returns the secret for tool+credential_id (auto-pick when id empty and exactly one match).
func (b *CredentialBroker) ResolveSecret(tool, credentialID string) (cred Credential, secret string, err error) {
	if b == nil || !b.Unlocked() {
		return Credential{}, "", fmt.Errorf("vault locked or unavailable")
	}
	tool = strings.TrimSpace(tool)
	credentialID = strings.TrimSpace(credentialID)
	b.mu.RLock()
	defer b.mu.RUnlock()
	var matches []Credential
	for _, c := range b.creds {
		if credentialAllowsTool(c, tool) {
			matches = append(matches, c)
		}
	}
	if credentialID != "" {
		for _, c := range matches {
			if c.ID == credentialID {
				if c.Secret == "" {
					return Credential{}, "", fmt.Errorf("credential has empty secret")
				}
				return c, c.Secret, nil
			}
		}
		return Credential{}, "", fmt.Errorf("credential %q not allowed for tool %q", credentialID, tool)
	}
	if len(matches) == 0 {
		return Credential{}, "", nil // no credential required/available
	}
	if len(matches) == 1 {
		c := matches[0]
		return c, c.Secret, nil
	}
	return Credential{}, "", fmt.Errorf("multiple credentials for %q; pass credential_id", tool)
}

func credentialAllowsTool(c Credential, tool string) bool {
	for _, t := range c.AllowedTools {
		if t == tool {
			return true
		}
	}
	return false
}

func normalizeToolList(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func defaultInjectArg(typ string) string {
	switch typ {
	case "api_key":
		return "api_key"
	case "token":
		return "token"
	default:
		return "password"
	}
}

var injectArgBlocklist = map[string]bool{
	"url": true, "uri": true, "host": true, "hostname": true, "endpoint": true,
	"path": true, "command": true, "args": true, "cwd": true, "base_url": true,
	"recipient": true, "to": true, "message": true, "text": true, "body": true,
	"credential_id": true,
}

func validateInjectArg(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("inject_arg required")
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return fmt.Errorf("invalid inject_arg")
	}
	if injectArgBlocklist[strings.ToLower(name)] {
		return fmt.Errorf("inject_arg %q is not allowed (not an auth slot)", name)
	}
	return nil
}

// prepareToolArgs strips credential_id, resolves secret, injects into inject_arg for MCP.
// Returns sanitized args JSON for the connector and a secret for redaction (may be empty).
func (b *CredentialBroker) prepareToolArgs(toolName, argumentsJSON string) (outArgs string, secret string, err error) {
	args := map[string]any{}
	if strings.TrimSpace(argumentsJSON) != "" {
		if e := json.Unmarshal([]byte(argumentsJSON), &args); e != nil {
			return argumentsJSON, "", nil // let MCP report invalid JSON
		}
	}
	credID := ""
	if v, ok := args["credential_id"]; ok {
		credID, _ = v.(string)
		delete(args, "credential_id")
	}
	cred, sec, err := b.ResolveSecret(toolName, credID)
	if err != nil {
		return "", "", err
	}
	if sec == "" {
		b2, _ := json.Marshal(args)
		return string(b2), "", nil
	}
	argName := cred.InjectArg
	if argName == "" {
		argName = defaultInjectArg(cred.Type)
	}
	if err := validateInjectArg(argName); err != nil {
		return "", "", err
	}
	args[argName] = sec
	b2, err := json.Marshal(args)
	if err != nil {
		return "", "", err
	}
	return string(b2), sec, nil
}

func redactSecret(text, secret string) string {
	if secret == "" || text == "" {
		return text
	}
	return strings.ReplaceAll(text, secret, "[redacted]")
}

// TryUnlockFromEnv unlocks when vault exists and KEIMLING_VAULT_PASSWORD(_FILE) is set.
func (b *CredentialBroker) TryUnlockFromEnv() error {
	if b == nil {
		return nil
	}
	if _, err := os.Stat(b.path); err != nil {
		return nil
	}
	pw, err := vaultPasswordFromEnv()
	if err != nil {
		return err
	}
	if pw == "" {
		return nil
	}
	return b.Unlock(pw)
}

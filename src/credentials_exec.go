package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

var keimlingCredPlaceholder = regexp.MustCompile(`\{\{keimling:([A-Za-z0-9._-]+)\}\}`)

type execCredPlan struct {
	Command      string            // placeholders resolved to $ENV or removed
	Env          map[string]string // inject_env -> secret
	EnvNames     []string          // names only (for review)
	SSHKeys      []Credential      // ssh_key secrets for agent bridge
	Secrets      []string          // for redaction
	NeedsExfil   bool              // true when Env is non-empty
	Placeholders []string
}

func parseCredPlaceholders(command string) []string {
	matches := keimlingCredPlaceholder.FindAllStringSubmatch(command, -1)
	seen := map[string]bool{}
	var ids []string
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		id := m[1]
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

func (b *CredentialBroker) planExecCredentials(command string) (execCredPlan, error) {
	plan := execCredPlan{Command: command, Env: map[string]string{}}
	if b == nil || !b.Unlocked() {
		if len(parseCredPlaceholders(command)) > 0 {
			return plan, fmt.Errorf("vault locked or unavailable")
		}
		return plan, nil
	}
	ids := parseCredPlaceholders(command)
	plan.Placeholders = ids
	if len(ids) == 0 {
		return plan, nil
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	resolved := command
	for _, id := range ids {
		c := credentialByID(b.creds, id)
		if c == nil {
			return plan, fmt.Errorf("unknown credential %q", id)
		}
		if !credentialAllowsTool(*c, "exec") {
			return plan, fmt.Errorf("credential %q not allowed for exec", id)
		}
		if strings.TrimSpace(c.Secret) == "" {
			return plan, fmt.Errorf("credential %q has empty secret", id)
		}
		plan.Secrets = append(plan.Secrets, c.Secret)
		ph := "{{keimling:" + id + "}}"
		switch c.Type {
		case "ssh_key":
			plan.SSHKeys = append(plan.SSHKeys, *c)
			resolved = strings.ReplaceAll(resolved, ph, "")
		default:
			envName := strings.TrimSpace(c.InjectEnv)
			if envName == "" {
				return plan, fmt.Errorf("credential %q has no inject_env", id)
			}
			if err := validateInjectEnv(envName); err != nil {
				return plan, err
			}
			if prev, ok := plan.Env[envName]; ok && prev != c.Secret {
				return plan, fmt.Errorf("inject_env %q used by multiple different secrets", envName)
			}
			plan.Env[envName] = c.Secret
			plan.EnvNames = appendUnique(plan.EnvNames, envName)
			resolved = strings.ReplaceAll(resolved, ph, "$"+envName)
			plan.NeedsExfil = true
		}
	}
	plan.Command = strings.TrimSpace(resolved)
	return plan, nil
}

func credentialByID(list []Credential, id string) *Credential {
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	return nil
}

func appendUnique(in []string, v string) []string {
	for _, x := range in {
		if x == v {
			return in
		}
	}
	return append(in, v)
}

var injectEnvBlocklist = map[string]bool{
	"PATH": true, "LD_PRELOAD": true, "LD_LIBRARY_PATH": true, "DYLD_INSERT_LIBRARIES": true,
	"HOME": true, "USER": true, "LOGNAME": true, "SHELL": true, "TERM": true,
	"SSH_AUTH_SOCK": true, "SSH_AGENT_PID": true,
	"KEIMLING_VAULT_PASSWORD": true, "KEIMLING_SECRET": true, "KEIMLING_ACCESS_SECRET": true,
	"OPENAI_API_KEY": true,
}

func validateInjectEnv(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("inject_env required")
	}
	for i, r := range name {
		if i == 0 {
			if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_') {
				return fmt.Errorf("invalid inject_env")
			}
			continue
		}
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return fmt.Errorf("invalid inject_env")
	}
	if injectEnvBlocklist[name] || injectEnvBlocklist[strings.ToUpper(name)] {
		return fmt.Errorf("inject_env %q is not allowed", name)
	}
	return nil
}

func mergeExecEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	out := append([]string{}, base...)
	for k, v := range extra {
		prefix := k + "="
		replaced := false
		for i, e := range out {
			if strings.HasPrefix(e, prefix) {
				out[i] = prefix + v
				replaced = true
				break
			}
		}
		if !replaced {
			out = append(out, prefix+v)
		}
	}
	return out
}

func redactAllSecrets(text string, secrets []string) string {
	for _, s := range secrets {
		text = redactSecret(text, s)
	}
	return text
}

// reviewExecSecretExfil asks the model whether injecting the named envs into command is safe.
func (c *Client) reviewExecSecretExfil(ctx context.Context, model, command string, envNames []string, credSummaries []string) (Verdict, error) {
	if c == nil {
		return Verdict{}, fmt.Errorf("no client")
	}
	if model == "" {
		model = "gpt-5"
	}
	instructions := `You review shell commands before Keimling injects secrets into environment variables.
Approve only if the command will NOT exfiltrate, log, echo, print, upload, or write those secrets to untrusted destinations.
Deny (ask) for: curl/wget/nc to remote with env secrets, echo/printf of secrets, redirecting secrets to world-readable files, shipping secrets in git commits, env|grep of secret names to remote, etc.
Normal use of secrets as intended auth (git over HTTPS with token in env consumed by git, API CLI reading the env) may be approved.
Return JSON only.`
	user := fmt.Sprintf("command=%q\nenv_names_to_inject=%q\ncredentials=%q\nReturn {\"decision\":\"approve\" or \"ask\",\"reason\":\"short English reason\"}.",
		command, strings.Join(envNames, ","), strings.Join(credSummaries, "; "))
	// Reuse Review transport shape via a thin dedicated call.
	return c.reviewJSONDecision(ctx, model, instructions, user)
}

package main

import (
	"strings"
)

// credentialStatusPrompt states the vault and git facts of this moment so the
// model does not rediscover them with probing commands. It never contains a secret.
func credentialStatusPrompt(b *CredentialBroker) string {
	if b == nil {
		return ""
	}
	var parts []string
	switch {
	case !b.Exists():
		parts = append(parts, "Credential vault: none stored on this host. Secrets that are not in the environment are unavailable; say so instead of guessing a key path.")
	case !b.Unlocked():
		parts = append(parts, "Credential vault: locked. Stored secrets and git over SSH fail until the user unlocks the vault in the UI. Ask once for the unlock, then wait; do not probe ~/.ssh or alternate identities meanwhile.")
	default:
		var execIDs []string
		for _, c := range b.ListPublic() {
			id, _ := c["id"].(string)
			typ, _ := c["type"].(string)
			if id == "" || !credentialAllowsTool(Credential{AllowedTools: stringSlice(c["allowed_tools"]), Type: typ}, "exec") {
				continue
			}
			if typ == "ssh_key" {
				execIDs = append(execIDs, id+" (ssh_key)")
				continue
			}
			execIDs = append(execIDs, id+" ("+typ+")")
		}
		if len(execIDs) == 0 {
			parts = append(parts, "Credential vault: unlocked, with no credential allowed for exec.")
		} else {
			parts = append(parts, "Credential vault: unlocked. Credentials for exec: "+strings.Join(execIDs, ", ")+". Put the id inline in the command as {{keimling:<id>}}; no other call is needed to load it.")
			for _, c := range b.ListPublic() {
				if typ, _ := c["type"].(string); typ != "ssh_key" {
					continue
				}
				if id, _ := c["id"].(string); id != "" {
					parts = append(parts, "Git over SSH: run `git pull` or `git push` with that placeholder in the command, for example `git pull {{keimling:"+id+"}}`. The harness starts an ephemeral ssh-agent, so git needs no extra SSH setup; do not edit ssh config, pass -i, or set GIT_SSH_COMMAND.")
					break
				}
			}
		}
	}
	return strings.Join(parts, " ")
}

func stringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

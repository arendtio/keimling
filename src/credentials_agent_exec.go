package main

import (
	"context"
	"encoding/json"
)

// runExecWithCredentials handles exec: placeholders, env inject (+ exfil review), ssh-agent.
func (a *Agent) runExecWithCredentials(ctx context.Context, argsJSON string) string {
	var x struct {
		Command string `json:"command"`
	}
	_ = json.Unmarshal([]byte(argsJSON), &x)
	command := x.Command
	if a == nil || a.Credentials == nil || len(parseCredPlaceholders(command)) == 0 {
		return executeTool(ctx, "exec", argsJSON)
	}
	plan, err := a.Credentials.planExecCredentials(command)
	if err != nil {
		return "credential error: " + err.Error()
	}
	// Rewrite command in args for execution (placeholders resolved).
	var argsMap map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &argsMap); err != nil {
		return err.Error()
	}
	argsMap["command"] = plan.Command
	rewritten, _ := json.Marshal(argsMap)

	if plan.NeedsExfil {
		summaries := make([]string, 0, len(plan.Placeholders))
		for _, id := range plan.Placeholders {
			summaries = append(summaries, id)
		}
		model := a.Cfg.ReviewModel
		if model == "" {
			model = a.Cfg.Model
		}
		verdict, rerr := a.Client.reviewExecSecretExfil(ctx, model, plan.Command, plan.EnvNames, summaries)
		if rerr != nil {
			return "credential exfil review failed (fail-closed): " + rerr.Error()
		}
		if verdict.Decision != "approve" {
			return "credential exfil review denied: " + verdict.Reason
		}
		a.println("◌ Credential env inject approved by exfil review")
	}

	var extraEnv []string
	for k, v := range plan.Env {
		extraEnv = append(extraEnv, k+"="+v)
	}
	var agentSess *sshAgentSession
	if len(plan.SSHKeys) > 0 {
		agentSess, err = startSSHAgentWithKeys(plan.SSHKeys)
		if err != nil {
			return "ssh-agent error: " + err.Error()
		}
		defer agentSess.Close()
		extraEnv = append(extraEnv, agentSess.Environ()...)
		a.println("◌ SSH agent loaded for credential inject")
	}

	result := executeToolWithEnv(ctx, "exec", string(rewritten), extraEnv)
	return redactAllSecrets(result, plan.Secrets)
}

func parseExecSecretsForRedact(a *Agent, argsJSON string) []string {
	if a == nil || a.Credentials == nil {
		return nil
	}
	var x struct {
		Command string `json:"command"`
	}
	_ = json.Unmarshal([]byte(argsJSON), &x)
	plan, err := a.Credentials.planExecCredentials(x.Command)
	if err != nil {
		return nil
	}
	return plan.Secrets
}

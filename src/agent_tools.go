package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

func (a *Agent) agentExtraTools() []map[string]any {
	if a == nil || strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return nil
	}
	return []map[string]any{
		toolDef("change_workspace", "Switch this agent instance to another allowed workspace (like cd). workspace_id must be in the instance ACL.", map[string]any{"workspace_id": map[string]any{"type": "string", "description": "Workspace id from the instance ACL"}}, "workspace_id"),
		toolDef("set_alarm", "Schedule a wake for this agent. when: relative \"in 2m\"/\"5m\", repeating \"every 30s\", clock \"at 15:04\", or RFC3339. message is injected as a user turn when the alarm fires. For repeating work use every N.", map[string]any{"when": map[string]any{"type": "string"}, "message": map[string]any{"type": "string"}}, "when", "message"),
		toolDef("list_alarms", "List pending alarms for this agent instance.", map[string]any{"unused": map[string]any{"type": "string", "description": "Pass empty string"}}, "unused"),
		toolDef("cancel_alarm", "Cancel a pending alarm by id from list_alarms.", map[string]any{"id": map[string]any{"type": "string"}}, "id"),
		toolDef("list_agents", "List other agent instances that share a workspace with you, plus your peer ACL. Use ids with allow_peer / message_agent.", map[string]any{"unused": map[string]any{"type": "string", "description": "Pass empty string"}}, "unused"),
		toolDef("allow_peer", "Add another agent instance id to this agent's peer ACL so message_agent may contact it.", map[string]any{"instance_id": map[string]any{"type": "string"}}, "instance_id"),
		toolDef("message_agent", "Send a text message to another agent instance's main thread (must be in peer ACL). Delivered via durable queue; wakes the peer when idle.", map[string]any{"instance_id": map[string]any{"type": "string"}, "text": map[string]any{"type": "string"}}, "instance_id", "text"),
	}
}

func (a *Agent) changeWorkspace(argsJSON string) string {
	if strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return "change_workspace error: only available on agent instances"
	}
	var x struct {
		WorkspaceID string `json:"workspace_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &x); err != nil || strings.TrimSpace(x.WorkspaceID) == "" {
		return "change_workspace error: workspace_id required"
	}
	id := strings.TrimSpace(x.WorkspaceID)
	allowed := false
	for _, w := range a.Cfg.AllowedWorkspaces {
		if w == id {
			allowed = true
			break
		}
	}
	if !allowed {
		return "change_workspace error: workspace not in ACL"
	}
	meta, err := readWorkspaceMeta(id)
	if err != nil {
		return "change_workspace error: " + err.Error()
	}
	if err := os.Chdir(meta.Root); err != nil {
		return "change_workspace error: " + err.Error()
	}
	if err := setAgentInstanceWorkspace(a.Cfg.AgentInstanceID, id); err != nil {
		return "change_workspace error: " + err.Error()
	}
	a.Cfg.WorkspaceID = id
	a.Cfg.WorkspaceRoot = meta.Root
	a.println("\033[36m▣ Workspace\033[0m " + meta.Name + " → " + meta.Root)
	return fmt.Sprintf("changed workspace to %s (%s)", id, meta.Root)
}

func (a *Agent) setAlarm(argsJSON string) string {
	if strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return "set_alarm error: only available on agent instances"
	}
	var x struct {
		When    string `json:"when"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &x); err != nil {
		return "set_alarm error: invalid arguments"
	}
	fireAt, every, err := parseAlarmWhen(x.When, time.Now())
	if err != nil {
		return "set_alarm error: " + err.Error()
	}
	al, err := addAgentAlarm(a.Cfg.AgentInstanceID, fireAt, x.Message, every)
	if err != nil {
		return "set_alarm error: " + err.Error()
	}
	if every > 0 {
		return fmt.Sprintf("alarm %s set for %s (repeats every %s)", al.ID, al.FireAt.Local().Format(time.RFC3339), every)
	}
	return fmt.Sprintf("alarm %s set for %s", al.ID, al.FireAt.Local().Format(time.RFC3339))
}

func (a *Agent) listAlarmsTool(_ string) string {
	if strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return "list_alarms error: only available on agent instances"
	}
	list, err := readAgentAlarms(a.Cfg.AgentInstanceID)
	if err != nil {
		return "list_alarms error: " + err.Error()
	}
	if len(list) == 0 {
		return "no pending alarms"
	}
	var b strings.Builder
	for _, al := range list {
		extra := ""
		if al.Every > 0 {
			extra = " every " + al.Every.String()
		} else if al.EveryRaw != "" {
			extra = " every " + al.EveryRaw
		}
		fmt.Fprintf(&b, "- %s at %s%s: %s\n", al.ID, al.FireAt.Local().Format(time.RFC3339), extra, al.Message)
	}
	return strings.TrimSpace(b.String())
}

func (a *Agent) cancelAlarm(argsJSON string) string {
	if strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return "cancel_alarm error: only available on agent instances"
	}
	var x struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &x); err != nil || strings.TrimSpace(x.ID) == "" {
		return "cancel_alarm error: id required"
	}
	if err := cancelAgentAlarm(a.Cfg.AgentInstanceID, x.ID); err != nil {
		return "cancel_alarm error: " + err.Error()
	}
	return "alarm cancelled"
}

func (a *Agent) listAgentsTool(_ string) string {
	if strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return "list_agents error: only available on agent instances"
	}
	self, err := readAgentInstanceMeta(a.Cfg.AgentInstanceID)
	if err != nil {
		return "list_agents error: " + err.Error()
	}
	all, err := listAgentInstances()
	if err != nil {
		return "list_agents error: " + err.Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "self: %s (%s)\npeers: %s\n", self.ID, self.Name, strings.Join(self.Peers, ", "))
	for _, m := range all {
		if m.ID == self.ID {
			continue
		}
		share := false
		for _, w := range m.Workspaces {
			if agentInstanceAllowsWorkspace(self, w) {
				share = true
				break
			}
		}
		peer := agentInstanceAllowsPeer(self, m.ID)
		if !share && !peer {
			continue
		}
		fmt.Fprintf(&b, "- %s name=%q recipe=%s peer_allowed=%v shared_workspace=%v\n", m.ID, m.Name, m.RecipeID, peer, share)
	}
	return strings.TrimSpace(b.String())
}

func (a *Agent) allowPeer(argsJSON string) string {
	if strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return "allow_peer error: only available on agent instances"
	}
	var x struct {
		InstanceID string `json:"instance_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &x); err != nil || strings.TrimSpace(x.InstanceID) == "" {
		return "allow_peer error: instance_id required"
	}
	if err := addAgentInstancePeer(a.Cfg.AgentInstanceID, x.InstanceID); err != nil {
		return "allow_peer error: " + err.Error()
	}
	return "peer allowed: " + strings.TrimSpace(x.InstanceID)
}

func (a *Agent) messageAgent(argsJSON string) string {
	if strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return "message_agent error: only available on agent instances"
	}
	var x struct {
		InstanceID string `json:"instance_id"`
		Text       string `json:"text"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &x); err != nil {
		return "message_agent error: invalid arguments"
	}
	target := strings.TrimSpace(x.InstanceID)
	text := strings.TrimSpace(x.Text)
	if target == "" || text == "" {
		return "message_agent error: instance_id and text required"
	}
	self, err := readAgentInstanceMeta(a.Cfg.AgentInstanceID)
	if err != nil {
		return "message_agent error: " + err.Error()
	}
	if !agentInstanceAllowsPeer(self, target) {
		return "message_agent error: peer not in ACL (use allow_peer first)"
	}
	if err := enqueueAgentMessage(target, text, a.Cfg.AgentInstanceID, "peer"); err != nil {
		return "message_agent error: " + err.Error()
	}
	return fmt.Sprintf("queued message for %s", target)
}

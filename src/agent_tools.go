package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func (a *Agent) credentialToolDefs() []map[string]any {
	return []map[string]any{
		toolDef("list_credentials", "List Keimling vault credentials allowed for a tool. Returns metadata only (never secrets). Pass the exact tool name (e.g. exec or mcp__server__tool). For exec, use placeholders {{keimling:<id>}} in the command; inject_env names come from the vault; type ssh_key uses an ephemeral ssh-agent.", map[string]any{"tool": map[string]any{"type": "string", "description": "Tool name to list credentials for"}}, "tool"),
	}
}

func (a *Agent) listCredentialsTool(argsJSON string) string {
	var x struct {
		Tool string `json:"tool"`
	}
	if err := decodeArgs(argsJSON, &x); err != nil {
		return "list_credentials error: " + err.Error()
	}
	if a == nil || a.Credentials == nil {
		return "list_credentials error: vault unavailable"
	}
	list, err := a.Credentials.ListForTool(x.Tool)
	if err != nil {
		return "list_credentials error: " + err.Error()
	}
	b, err := json.Marshal(map[string]any{"credentials": list})
	if err != nil {
		return "list_credentials error: " + err.Error()
	}
	return string(b)
}

func (a *Agent) agentExtraTools() []map[string]any {
	if a == nil || strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return nil
	}
	return []map[string]any{
		toolDef("change_workspace", "Switch this agent instance to another allowed workspace (like cd). workspace_id must be in the instance ACL.", map[string]any{"workspace_id": map[string]any{"type": "string", "description": "Workspace id from the instance ACL"}}, "workspace_id"),
		toolDef("set_alarm", "Schedule a wake for this agent only. Another agent cannot set this alarm, and this alarm does not wake them. when: relative \"in 2m\"/\"5m\", repeating \"every 30s\", clock \"at 15:04\", or RFC3339. message is a reminder to yourself, shown as [Alarm] when it fires. It is not sent to the user or to another agent. For repeating work use every N.", map[string]any{"when": map[string]any{"type": "string"}, "message": map[string]any{"type": "string", "description": "Reminder for yourself when this alarm fires. Not a message to the user or to another agent."}}, "when", "message"),
		toolDef("list_alarms", "List pending alarms for this agent instance.", map[string]any{"unused": map[string]any{"type": "string", "description": "Pass empty string"}}, "unused"),
		toolDef("cancel_alarm", "Cancel a pending alarm by id from list_alarms.", map[string]any{"id": map[string]any{"type": "string"}}, "id"),
		toolDef("list_agents", "List other agent instances the user has allowed this agent to contact. Pass one of those instance_id values to message_agent. Never pass your own id. You cannot add contacts yourself. A thread_id from an incoming message is also a valid message_agent target.", map[string]any{"unused": map[string]any{"type": "string", "description": "Pass empty string"}}, "unused"),
		toolDef("message_agent", "Send a text message to one recipient. instance_id is another agent's instance id from list_agents, or the thread_id copied exactly from an incoming [From thread_id=... via message_agent] line. Do not add a prefix such as agent-. Never pass your own id. text must stand alone: the recipient does not see your chat, alarms, or recent events. Writing in this chat does not deliver it. A line marked via message_agent is answered with this tool, not with assistant text. If another agent asked you for something, the answer goes back to that agent. Do not tell them to deliver it to the user. The user sets which agents may contact each other. A thread in a workspace this agent may use can be addressed without that contact list.", map[string]any{"instance_id": map[string]any{"type": "string", "description": "The other agent's instance id, or a thread_id copied exactly. Not your own id. Do not add a prefix."}, "text": map[string]any{"type": "string", "description": "The message for that recipient. Name what you mean; they do not have your context. A reply goes back to the sender, not to the user."}}, "instance_id", "text"),
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
	self, err := readAgentInstanceMeta(a.Cfg.AgentInstanceID)
	if err != nil {
		return "change_workspace error: " + err.Error()
	}
	a.Cfg.AllowedWorkspaces = append([]string(nil), self.Workspaces...)
	if !agentInstanceAllowsWorkspace(self, id) {
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
	fmt.Fprintf(&b, "You are %s. Your instance id is %s. Do not pass your own id to message_agent.\n", self.Name, self.ID)
	fmt.Fprintf(&b, "Pass one of these instance_id values:\n")
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
		peer := agentsMayContact(self, m)
		if !share && !peer {
			continue
		}
		fmt.Fprintf(&b, "- instance_id=%s name=%q peer_allowed=%v shared_workspace=%v\n", m.ID, m.Name, peer, share)
	}
	return strings.TrimSpace(b.String())
}

func (a *Agent) allowPeer(string) string {
	if strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return "allow_peer error: only available on agent instances"
	}
	return "allow_peer error: the user sets contacts in Manage agents"
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
	if wsID, threadID, title, ok := resolveThreadAddress(self, target); ok {
		return a.deliverThreadMessage(self, wsID, threadID, title, text)
	}
	targetMeta, err := resolveAgentInstance(target)
	if err != nil && strings.HasPrefix(target, "agent-") {
		if wsID, threadID, title, ok := resolveThreadAddress(self, strings.TrimPrefix(target, "agent-")); ok {
			return a.deliverThreadMessage(self, wsID, threadID, title, text)
		}
	}
	if err != nil {
		return "message_agent error: unknown agent or thread"
	}
	if targetMeta.ID == self.ID {
		return "message_agent error: instance_id is this agent (" + self.Name + "). Pass the other agent's instance_id from list_agents, or a thread_id."
	}
	if !agentsMayContact(self, targetMeta) {
		return "message_agent error: the user has not allowed contact with " + targetMeta.Name + " (" + targetMeta.ID + ")"
	}
	target = targetMeta.ID
	if err := enqueueAgentMessage(target, text, a.Cfg.AgentInstanceID, "peer"); err != nil {
		return "message_agent error: " + err.Error()
	}
	name := agentInstanceDisplayName(target)
	a.recordPeerSent(name, text)
	return fmt.Sprintf("Sent to %s: %s", name, text)
}

func (a *Agent) deliverThreadMessage(self AgentInstanceMeta, workspaceID, threadID, title, text string) string {
	if err := deliverAgentTextToThread(self, workspaceID, threadID, text); err != nil {
		return "message_agent error: " + err.Error()
	}
	a.recordPeerSent(title, text)
	return fmt.Sprintf("Sent to thread %s: %s", title, text)
}

func resolveThreadAddress(self AgentInstanceMeta, ref string) (workspaceID, threadID, title string, ok bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", "", false
	}
	seen := map[string]bool{}
	workspaces := append([]string{}, self.Workspaces...)
	if cur := strings.TrimSpace(self.CurrentWorkspace); cur != "" {
		workspaces = append(workspaces, cur)
	}
	for _, ws := range workspaces {
		ws = strings.TrimSpace(ws)
		if ws == "" || seen[ws] {
			continue
		}
		seen[ws] = true
		st, err := os.Stat(threadDir(ws, ref))
		if err != nil || !st.IsDir() {
			continue
		}
		title = ref
		if meta, err := readThreadMeta(ws, ref); err == nil && strings.TrimSpace(meta.Title) != "" {
			title = strings.TrimSpace(meta.Title)
		}
		return ws, ref, title, true
	}
	return "", "", "", false
}

func deliverAgentTextToThread(from AgentInstanceMeta, workspaceID, threadID, text string) error {
	name := strings.TrimSpace(from.Name)
	if name == "" {
		name = from.ID
	}
	if postThreadDeliver(workspaceID, threadID, name, from.ID, text) == nil {
		return nil
	}
	return appendSessionItem(threadSessionPath(workspaceID, threadID), map[string]any{
		"type":      "keimling_peer",
		"direction": "received",
		"peer":      name,
		"text":      text,
	})
}

func postThreadDeliver(workspaceID, threadID, fromName, fromID, text string) error {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("KEIMLING_HOST")), "/")
	secret := strings.TrimSpace(os.Getenv("KEIMLING_SECRET"))
	if base == "" || secret == "" {
		return fmt.Errorf("no host")
	}
	body, err := json.Marshal(map[string]string{
		"workspace_id": workspaceID,
		"thread_id":    threadID,
		"from_name":    fromName,
		"from_id":      fromID,
		"text":         text,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, base+"/api/threads/deliver", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("deliver HTTP %d", res.StatusCode)
	}
	return nil
}

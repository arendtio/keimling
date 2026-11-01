package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func rawItem(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func TestPeerReplyHintNamesTheRecipient(t *testing.T) {
	const (
		thread = "20261031-225911-490222439"
		inst   = "agent-20261101-005219-472607549"
		ent    = "agent-20261101-005219-472142000"
		joke   = "Why don't scientists trust atoms? Because they make up everything!"
	)
	ask := rawItem(map[string]any{"role": "user", "content": "[From thread_id=" + thread + " via message_agent] Ask the Entertainer for a joke and tell it to me."})
	jokeReq := rawItem(map[string]any{"role": "user", "content": "[From Instructor 15 instance_id=" + inst + " via message_agent] Tell me a joke."})
	listCall := rawItem(map[string]any{"type": "function_call", "name": "list_agents", "call_id": "call_list", "arguments": `{"unused":""}`})
	listOut := rawItem(map[string]any{"type": "function_call_output", "call_id": "call_list", "output": "You are Instructor. Your instance id is " + inst + ".\n- instance_id=" + ent + " name=\"Entertainer\" peer_allowed=true shared_workspace=true\n"})
	askCall := rawItem(map[string]any{"type": "function_call", "name": "message_agent", "call_id": "call_ask", "arguments": `{"instance_id":"` + ent + `","text":"Tell me a joke."}`})
	askOut := rawItem(map[string]any{"type": "function_call_output", "call_id": "call_ask", "output": "Sent to Entertainer: Tell me a joke."})
	jokeIn := rawItem(map[string]any{"role": "user", "content": "[From Entertainer 15 instance_id=" + ent + " via message_agent] " + joke})
	backCall := rawItem(map[string]any{"type": "function_call", "name": "message_agent", "call_id": "call_back", "arguments": `{"instance_id":"` + thread + `","text":"` + joke + `"}`})
	backOut := rawItem(map[string]any{"type": "function_call_output", "call_id": "call_back", "output": "Sent to thread: " + joke})
	did := rawItem(map[string]any{"role": "user", "content": "[From thread_id=" + thread + " via message_agent] Did you tell me the joke?"})

	if hint := peerReplyHint([]json.RawMessage{ask}, inst); hint != "" {
		t.Fatalf("first ask should list agents before a reply hint, got %q", hint)
	}
	answer := peerReplyHint([]json.RawMessage{jokeReq}, ent)
	if !strings.Contains(answer, "instance_id: "+inst) || !strings.Contains(answer, "your answer to the message above") {
		t.Fatalf("direct request hint: %q", answer)
	}
	asked := peerReplyHint([]json.RawMessage{ask, listCall, listOut}, inst)
	if !strings.Contains(asked, "instance_id: "+ent) || !strings.Contains(asked, "text is: Tell me a joke.") {
		t.Fatalf("pass-on hint: %q", asked)
	}
	relay := peerReplyHint([]json.RawMessage{ask, listCall, listOut, askCall, askOut, jokeIn}, inst)
	if !strings.Contains(relay, "instance_id: "+thread) || !strings.Contains(relay, "text: "+joke) {
		t.Fatalf("relay hint: %q", relay)
	}
	follow := peerReplyHint([]json.RawMessage{ask, listCall, listOut, askCall, askOut, jokeIn, backCall, backOut, did}, inst)
	if !strings.Contains(follow, "instance_id: "+thread) || !strings.Contains(follow, "your answer to the message above") {
		t.Fatalf("follow-up hint: %q", follow)
	}
}

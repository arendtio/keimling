package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

const encryptedReasoningOmittedNotice = "Encrypted reasoning is not sent to this model. It stays in the thread and is included again when you switch back."

const (
	reasoningProviderField = "keimling_provider"
	reasoningModelField    = "keimling_model"
)

// modelPair is the provider and model that produced an encrypted reasoning blob.
type modelPair struct {
	Provider string
	Model    string
}

func (p modelPair) same(o modelPair) bool {
	return p.Provider == o.Provider && p.Model == o.Model
}

type reasoningOwner struct {
	Key        string `json:"key"`
	ProviderID string `json:"provider_id"`
	ModelID    string `json:"model_id"`
}

// prepareReasoningInput omits encrypted reasoning that belongs to another model.
// Readable reasoning text is kept. Unknown blobs stay, so a thread that has not
// switched model still sends what it stored. The history itself is not edited.
func prepareReasoningInput(input []json.RawMessage, provider, model string, owners map[string]modelPair) []json.RawMessage {
	target := modelPair{Provider: provider, Model: model}
	out := make([]json.RawMessage, 0, len(input))
	for _, raw := range input {
		var item map[string]json.RawMessage
		if json.Unmarshal(raw, &item) != nil || !isReasoningItem(item) {
			out = append(out, raw)
			continue
		}
		blob := encryptedBlob(item)
		text := readableReasoning(item)
		if blob == "" && text == "" {
			continue
		}
		// Reasoning item content must be empty on the wire. OpenAI rejects a
		// non-empty array ("maximum length 0"). The readable text travels as
		// its own message, which both OpenAI and llama.cpp accept.
		if text != "" {
			if msg, err := reasoningTextMessage(text); err == nil {
				out = append(out, msg)
			}
		}
		owner, known := reasoningOwnerOf(item, owners)
		if blob != "" && (!known || owner.same(target)) {
			out = append(out, reasoningItemWithoutContent(raw))
		}
	}
	return out
}

func reasoningTextMessage(text string) (json.RawMessage, error) {
	return json.Marshal(map[string]string{
		"role":    "developer",
		"content": "Reasoning from an earlier model:\n" + text,
	})
}

func reasoningItemWithoutContent(raw json.RawMessage) json.RawMessage {
	var item map[string]json.RawMessage
	if json.Unmarshal(raw, &item) != nil || item == nil {
		return raw
	}
	delete(item, "content")
	stripped, err := json.Marshal(item)
	if err != nil {
		return raw
	}
	return stripped
}

func (a *Agent) annotateReasoning(raw json.RawMessage) json.RawMessage {
	if a == nil {
		return raw
	}
	t := a.callChatTarget()
	owners := a.reasoningOwnerMap()
	stamped, changed := stampReasoningOwner(raw, modelPair{Provider: t.ProviderID, Model: t.Model}, owners)
	if changed {
		a.persistReasoningOwners(owners)
		return stamped
	}
	return raw
}

// noteEncryptedReasoningOnModelChange tells the user at the moment the chat
// model changes that encrypted reasoning for another model will not be sent.
func (a *Agent) noteEncryptedReasoningOnModelChange(prev chatTarget) {
	if a == nil {
		return
	}
	next := a.effectiveChatTarget()
	if next.ProviderID == prev.ProviderID && next.Model == prev.Model {
		return
	}
	a.attributeUnownedReasoning(modelPair{Provider: prev.ProviderID, Model: prev.Model})
	if a.omitsEncryptedReasoning(next.ProviderID, next.Model) {
		a.note(encryptedReasoningOmittedNotice)
	}
}

func (a *Agent) attributeUnownedReasoning(owner modelPair) {
	if a == nil || owner.Model == "" {
		return
	}
	owners := a.reasoningOwnerMap()
	changed := false
	nextHist := make([]json.RawMessage, len(a.History))
	for i, raw := range a.History {
		stamped, did := stampReasoningOwner(raw, owner, owners)
		nextHist[i] = stamped
		if did {
			changed = true
		}
	}
	if changed {
		a.History = nextHist
		if a.Journal != nil {
			if err := a.Journal.AppendCheckpoint(nextHist, "reasoning-owner"); err == nil {
				a.History = a.Journal.History
			}
		}
	}
	if len(owners) > 0 {
		a.persistReasoningOwners(owners)
	}
}

func (a *Agent) omitsEncryptedReasoning(provider, model string) bool {
	if a == nil {
		return false
	}
	target := modelPair{Provider: provider, Model: model}
	owners := a.reasoningOwnerMap()
	for _, raw := range a.History {
		var item map[string]json.RawMessage
		if json.Unmarshal(raw, &item) != nil || !isReasoningItem(item) || encryptedBlob(item) == "" {
			continue
		}
		owner, known := reasoningOwnerOf(item, owners)
		if known && !owner.same(target) {
			return true
		}
	}
	return false
}

func stampReasoningOwner(raw json.RawMessage, owner modelPair, owners map[string]modelPair) (json.RawMessage, bool) {
	var item map[string]json.RawMessage
	if json.Unmarshal(raw, &item) != nil || !isReasoningItem(item) || encryptedBlob(item) == "" {
		return raw, false
	}
	if owners == nil {
		owners = map[string]modelPair{}
	}
	if existing, ok := ownerOnItem(item); ok {
		if key := reasoningItemKey(item); key != "" {
			owners[key] = existing
		}
		return raw, false
	}
	if owner.Model == "" {
		return raw, false
	}
	provider, _ := json.Marshal(owner.Provider)
	model, _ := json.Marshal(owner.Model)
	item[reasoningProviderField] = provider
	item[reasoningModelField] = model
	if key := reasoningItemKey(item); key != "" {
		owners[key] = owner
	}
	stamped, err := json.Marshal(item)
	if err != nil {
		return raw, false
	}
	return stamped, true
}

func reasoningOwnerOf(item map[string]json.RawMessage, owners map[string]modelPair) (modelPair, bool) {
	if owner, ok := ownerOnItem(item); ok {
		return owner, true
	}
	if owners == nil {
		return modelPair{}, false
	}
	key := reasoningItemKey(item)
	if key == "" {
		return modelPair{}, false
	}
	owner, ok := owners[key]
	if !ok || owner.Model == "" {
		return modelPair{}, false
	}
	return owner, true
}

func ownerOnItem(item map[string]json.RawMessage) (modelPair, bool) {
	model, ok := jsonStringField(item[reasoningModelField])
	if !ok || model == "" {
		return modelPair{}, false
	}
	provider, _ := jsonStringField(item[reasoningProviderField])
	return modelPair{Provider: provider, Model: model}, true
}

func reasoningItemKey(item map[string]json.RawMessage) string {
	if id, ok := jsonStringField(item["id"]); ok && id != "" {
		return id
	}
	blob := encryptedBlob(item)
	if blob == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(blob))
	return "blob:" + hex.EncodeToString(sum[:8])
}

func isReasoningItem(item map[string]json.RawMessage) bool {
	typ, ok := jsonStringField(item["type"])
	return ok && typ == "reasoning"
}

func encryptedBlob(item map[string]json.RawMessage) string {
	s, ok := jsonStringField(item["encrypted_content"])
	if !ok {
		return ""
	}
	return s
}

func readableReasoning(item map[string]json.RawMessage) string {
	parts := textParts(item["content"])
	parts = append(parts, textParts(item["summary"])...)
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func textParts(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	if s, ok := jsonStringField(raw); ok {
		if s == "" {
			return nil
		}
		return []string{s}
	}
	var arr []map[string]any
	if json.Unmarshal(raw, &arr) != nil {
		return nil
	}
	var out []string
	for _, part := range arr {
		text, _ := part["text"].(string)
		text = strings.TrimSpace(text)
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

func jsonStringField(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return "", false
	}
	return strings.TrimSpace(s), true
}

func (a *Agent) reasoningOwnerMap() map[string]modelPair {
	out := map[string]modelPair{}
	meta, err := a.readOwnThreadMeta()
	if err != nil {
		return out
	}
	for _, o := range meta.ReasoningOwners {
		if strings.TrimSpace(o.Key) == "" || strings.TrimSpace(o.ModelID) == "" {
			continue
		}
		out[o.Key] = modelPair{Provider: o.ProviderID, Model: o.ModelID}
	}
	return out
}

func (a *Agent) persistReasoningOwners(owners map[string]modelPair) {
	if a == nil {
		return
	}
	keys := make([]string, 0, len(owners))
	for key, owner := range owners {
		if key == "" || owner.Model == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	list := make([]reasoningOwner, 0, len(keys))
	for _, key := range keys {
		owner := owners[key]
		list = append(list, reasoningOwner{Key: key, ProviderID: owner.Provider, ModelID: owner.Model})
	}
	_ = a.updateOwnThreadMeta(func(meta *ThreadMeta) error {
		meta.ReasoningOwners = list
		return nil
	})
}

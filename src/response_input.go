package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// normalizeResponseInput converts persisted response OUTPUT objects to the
// minimal replayable INPUT shape. It never edits the original session journal.
// Status and server-assigned IDs belong to responses, not newly created input
// items on every Responses-compatible provider.
func normalizeResponseInput(history []json.RawMessage) ([]json.RawMessage, error) {
	result := make([]json.RawMessage, 0, len(history))
	for i, raw := range history {
		var item map[string]json.RawMessage
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, fmt.Errorf("history item %d: %w", i, err)
		}
		if item == nil {
			return nil, fmt.Errorf("history item %d: not an object", i)
		}
		var typ string
		_ = json.Unmarshal(item["type"], &typ)
		switch typ {
		case "message", "":
			// Preserve role, content and phase, but discard response-only metadata.
			delete(item, "status")
			delete(item, "id")
			delete(item, "annotations")
			if typ == "message" {
				// Output message content includes text annotations/logprobs,
				// which are not needed when replayed as message input.
				var parts []map[string]json.RawMessage
				if json.Unmarshal(item["content"], &parts) == nil {
					for _, part := range parts {
						delete(part, "annotations")
						delete(part, "logprobs")
					}
					item["content"], _ = json.Marshal(parts)
				}
			}
		case "function_call":
			item = keepInputFields(item, "type", "name", "call_id", "arguments")
		case "function_call_output":
			item = keepInputFields(item, "type", "call_id", "output")
		case "keimling_peer":
			var peer, text, direction string
			_ = json.Unmarshal(item["peer"], &peer)
			_ = json.Unmarshal(item["text"], &text)
			_ = json.Unmarshal(item["direction"], &direction)
			body, files := splitChatFiles(text)
			line := "Sent to " + strings.TrimSpace(peer) + ": " + strings.TrimSpace(body)
			if direction == "received" {
				line = "From " + strings.TrimSpace(peer) + ": " + strings.TrimSpace(body)
			}
			for _, f := range files {
				line += "\nfile: " + f.Path
			}
			role, _ := json.Marshal("developer")
			content, _ := json.Marshal(line)
			item = map[string]json.RawMessage{
				"role":    role,
				"content": content,
			}
		case "keimling_notice":
			var text string
			_ = json.Unmarshal(item["text"], &text)
			role, _ := json.Marshal("developer")
			content, _ := json.Marshal(strings.TrimSpace(text))
			item = map[string]json.RawMessage{
				"role":    role,
				"content": content,
			}
		case "reasoning":
			// encrypted_content continues stateless reasoning for the model that
			// produced it. A content array is not valid replay input: OpenAI
			// allows only an empty one. Readable text is added separately.
			// Ownership stamps are local and must not be sent.
			item = keepInputFields(item, "type", "id", "summary", "encrypted_content")
			if encryptedBlob(item) == "" {
				delete(item, "encrypted_content")
			}
		default:
			// Future or provider-specific items: remove only top-level response
			// metadata; never recursively modify embedded tool data.
			delete(item, "status")
		}
		normalized, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("history item %d: %w", i, err)
		}
		result = append(result, normalized)
	}
	return result, nil
}

func keepInputFields(item map[string]json.RawMessage, names ...string) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(names))
	for _, name := range names {
		if value, ok := item[name]; ok {
			out[name] = value
		}
	}
	return out
}

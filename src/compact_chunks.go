package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Each API request is bounded independently of total history size. Split at
// record boundaries where possible, but also handle huge single records.
const compactionChunkBytes = 120000
const compactionSummaryTokens = 2500

func compactMaterial(ctx context.Context, c *Client, model string, history []json.RawMessage, instruction string) (string, error) {
	chunks := make([]string, 0)
	var buf strings.Builder
	flush := func() {
		if buf.Len() > 0 {
			chunks = append(chunks, buf.String())
			buf.Reset()
		}
	}
	for _, raw := range history {
		entry := string(raw) + "\n"
		if buf.Len()+len(entry) > compactionChunkBytes {
			flush()
		}
		for len(entry) > compactionChunkBytes {
			chunks = append(chunks, entry[:compactionChunkBytes])
			entry = entry[compactionChunkBytes:]
		}
		buf.WriteString(entry)
	}
	flush()
	if len(chunks) == 0 {
		return "", fmt.Errorf("empty history")
	}
	// Map-reduce: summarize bounded pieces, then summarize their summaries.
	for depth := 0; depth < 12; depth++ {
		next := make([]string, 0, len(chunks))
		for i, chunk := range chunks {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			prompt := fmt.Sprintf("Compaction pass %d, segment %d/%d. Preserve exact operational facts and pending work. %s\n%s", depth+1, i+1, len(chunks), instruction, chunk)
			text, err := c.Summarize(ctx, model, prompt, compactionSummaryTokens)
			if err != nil {
				return "", fmt.Errorf("segment %d/%d: %w", i+1, len(chunks), err)
			}
			if strings.TrimSpace(text) == "" {
				return "", fmt.Errorf("empty segment summary")
			}
			next = append(next, text)
		}
		if len(next) == 1 {
			return strings.TrimSpace(next[0]), nil
		}
		chunks = nil
		buf.Reset()
		for i, part := range next {
			entry := fmt.Sprintf("[Segment %d]\n%s\n", i+1, part)
			if buf.Len()+len(entry) > compactionChunkBytes {
				flush()
			}
			for len(entry) > compactionChunkBytes {
				chunks = append(chunks, entry[:compactionChunkBytes])
				entry = entry[compactionChunkBytes:]
			}
			buf.WriteString(entry)
		}
		flush()
		if len(chunks) >= len(next) {
			return "", fmt.Errorf("compaction did not converge (%d sections)", len(chunks))
		}
	}
	return "", fmt.Errorf("compaction exceeded reduction passes")
}

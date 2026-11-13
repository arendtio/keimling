package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

const delegateMapLimit = 8

// delegateSourceBudget is about 80% of the current model's window, in bytes,
// at four bytes per token. The window is the only ceiling.
func delegateSourceBudget(limit int) int {
	if limit < 1000 {
		limit = 8000
	}
	n := int64(limit) * 4 * 8 / 10
	if n < 1 {
		return 1
	}
	return int(n)
}

func (a *Agent) runDelegateSource(ctx context.Context, args DelegateArgs) (string, error) {
	model, effort, err := delegateProfile(a.Cfg, args.Profile)
	if err != nil {
		return "", err
	}
	material, span, err := a.delegateMaterial(args.Source)
	if err != nil {
		return "", err
	}
	if delegateHasSecret(material) {
		return "", errors.New("delegate context contains secret material")
	}
	note := strings.TrimSpace(args.Context)
	budget := delegateSourceBudget(a.activeChatContextLimit())
	room := budget - len(args.Task) - len(note) - 64
	if room < 512 {
		room = 512
	}
	parts := packParts(material, room)
	total := len(parts)
	if span > int64(len(material)) {
		total += int((span - int64(len(material)) + int64(room) - 1) / int64(room))
	}
	if len(parts) > delegateMapLimit {
		parts = parts[:delegateMapLimit]
	}
	if len(parts) == 1 {
		call := args
		call.Image = ""
		call.Source = ""
		call.Context = joinDelegateNote(note, parts[0])
		answer, err := a.Client.Delegate(ctx, model, effort, call, nil)
		if err != nil {
			return "", err
		}
		if total > 1 {
			return fmt.Sprintf("Read 1 of %d parts.\n%s", total, strings.TrimSpace(answer)), nil
		}
		return answer, nil
	}
	var answers []string
	for i, part := range parts {
		call := DelegateArgs{
			Task:    args.Task,
			Context: joinDelegateNote(note, part),
			Profile: args.Profile,
		}
		answer, err := a.Client.Delegate(ctx, model, effort, call, nil)
		if err != nil {
			return "", err
		}
		answers = append(answers, fmt.Sprintf("%d. %s", i+1, strings.TrimSpace(answer)))
	}
	joined := strings.Join(answers, "\n")
	keep := budget - len(args.Task) - 64
	if keep < 1 {
		keep = 1
	}
	if len(joined) > keep {
		joined = string(trimPartialUTF8([]byte(joined[:keep])))
	}
	reduce := DelegateArgs{
		Task:    "Combine these partial answers into one answer to the question.",
		Context: "Question:\n" + args.Task + "\n\n" + joined,
		Profile: args.Profile,
	}
	answer, err := a.Client.Delegate(ctx, model, effort, reduce, nil)
	if err != nil {
		return "", err
	}
	if total > len(parts) {
		return fmt.Sprintf("Read %d of %d parts.\n%s", len(parts), total, strings.TrimSpace(answer)), nil
	}
	return answer, nil
}

func (a *Agent) delegateMaterial(source string) (string, int64, error) {
	ref := strings.TrimSpace(source)
	if ref != "" && !resultRef(ref) && !historySectionRef(ref) {
		return a.delegateFile(ref)
	}
	v, err := a.resolveDataAddress(ref)
	if err != nil {
		return "", 0, err
	}
	if s, ok := v.(string); ok {
		return s, int64(len(s)), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", 0, err
	}
	return string(b), int64(len(b)), nil
}

// delegateFile reads a path up to eight model windows. The read grows with the
// current model. span is the file size, which can be larger than the text read.
func (a *Agent) delegateFile(path string) (string, int64, error) {
	if privatePath(path) {
		return "", 0, errors.New("access denied")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", 0, err
	}
	if st.IsDir() {
		return "", 0, errors.New("path is a directory")
	}
	span := st.Size()
	room := delegateSourceBudget(a.activeChatContextLimit())
	if room < 1 {
		room = 1
	}
	n := span
	maxRead := int64(room) * int64(delegateMapLimit)
	if n > maxRead {
		n = maxRead
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(f, buf); err != nil {
		return "", 0, err
	}
	if n < span {
		buf = trimPartialUTF8(buf)
	}
	if !utf8.Valid(buf) {
		return "", 0, errors.New("not a UTF-8 text file")
	}
	return string(buf), span, nil
}

func trimPartialUTF8(buf []byte) []byte {
	for i := 0; i < utf8.UTFMax && len(buf) > 0 && !utf8.Valid(buf); i++ {
		buf = buf[:len(buf)-1]
	}
	return buf
}

func joinDelegateNote(note, body string) string {
	if note == "" {
		return body
	}
	return note + "\n\n" + body
}

func packParts(text string, room int) []string {
	if room < 512 {
		room = 512
	}
	if len(text) <= room {
		return []string{text}
	}
	records := sourceRecords(text)
	if len(records) <= 1 {
		return splitDelegateText(text, room)
	}
	var parts []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		parts = append(parts, cur.String())
		cur.Reset()
	}
	for _, rec := range records {
		if len(rec) > room {
			flush()
			parts = append(parts, splitDelegateText(rec, room)...)
			continue
		}
		if cur.Len() > 0 && cur.Len()+1+len(rec) > room {
			flush()
		}
		if cur.Len() > 0 {
			cur.WriteByte('\n')
		}
		cur.WriteString(rec)
	}
	flush()
	if len(parts) == 0 {
		return []string{text}
	}
	return parts
}

func sourceRecords(text string) []string {
	if v, err := parseJSONText(text); err == nil {
		if recs := jsonRecords(v); len(recs) > 1 {
			return recs
		}
	}
	if v, err := parseXMLText(text); err == nil {
		if recs := xmlRecords(v); len(recs) > 1 {
			return recs
		}
	}
	if looksLikeCSV(text) {
		if v, err := parseCSVText(text); err == nil {
			if recs := jsonRecords(v); len(recs) > 1 {
				return recs
			}
		}
	}
	lines := strings.Split(text, "\n")
	if len(lines) > 1 {
		return lines
	}
	return []string{text}
}

func jsonRecords(v any) []string {
	switch x := v.(type) {
	case []any:
		if len(x) < 2 {
			return nil
		}
		return marshalEach(x)
	case map[string]any:
		key := ""
		var arr []any
		for k, val := range x {
			items, ok := val.([]any)
			if !ok || len(items) < 2 {
				continue
			}
			if len(items) > len(arr) {
				key, arr = k, items
			}
		}
		if key == "" {
			return nil
		}
		var out []string
		for _, el := range arr {
			copy := make(map[string]any, len(x))
			for k, val := range x {
				copy[k] = val
			}
			copy[key] = []any{el}
			b, err := json.Marshal(copy)
			if err != nil {
				continue
			}
			out = append(out, string(b))
		}
		return out
	default:
		return nil
	}
}

func xmlRecords(v any) []string {
	node, ok := v.(map[string]any)
	if !ok || !isXMLNode(node) {
		return nil
	}
	kids := elementChildren(node)
	for len(kids) == 1 {
		next := elementChildren(kids[0])
		if len(next) == 0 {
			break
		}
		kids = next
	}
	if len(kids) < 2 {
		return nil
	}
	items := make([]any, len(kids))
	for i := range kids {
		items[i] = kids[i]
	}
	return marshalEach(items)
}

func elementChildren(node map[string]any) []map[string]any {
	children, _ := node["children"].([]any)
	var out []map[string]any
	for _, child := range children {
		m, ok := child.(map[string]any)
		if ok && isXMLNode(m) {
			out = append(out, m)
		}
	}
	return out
}

func marshalEach(items []any) []string {
	var out []string
	for _, el := range items {
		b, err := json.Marshal(el)
		if err != nil {
			continue
		}
		out = append(out, string(b))
	}
	return out
}

func splitDelegateText(text string, room int) []string {
	if room < 1 {
		room = 1
	}
	var out []string
	for len(text) > room {
		cut := room
		if i := strings.LastIndex(text[:cut], "\n"); i > room/2 {
			cut = i + 1
		}
		out = append(out, text[:cut])
		text = text[cut:]
	}
	if text != "" {
		out = append(out, text)
	}
	return out
}

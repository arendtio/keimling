package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	chatFileBegin   = "\n\n<keimling_files>\n"
	chatFileEnd     = "\n</keimling_files>"
	chatInlineLimit = 32 << 10
	chatImageLimit  = 4 << 20
	chatAudioLimit  = 8 << 20
	chatUploadLimit = 32 << 20
)

// imageDelegateHint tells the chat model how to look at an attached picture.
const imageDelegateHint = "Call delegate with image set to this path. task is the user's question in the user's words. Do not rewrite it into a detailed description. context is the earlier conversation that question depends on, or empty. This conversation does not include the image bytes. Call delegate again with the same path for another question."

// chatFile is a file a user message refers to. The path is what tools open.
type chatFile struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source string `json:"source"` // workspace | path | temp
}

func attachmentRoot() string {
	return filepath.Join(dataHome(), "keimling", "attachments")
}

func (m providerModel) nativeImages() bool {
	if !m.SeesImages {
		return false
	}
	if m.UseImages == nil {
		return true
	}
	return *m.UseImages
}

func (m providerModel) nativeAudio() bool {
	if !m.HearsAudio {
		return false
	}
	if m.UseAudio == nil {
		return true
	}
	return *m.UseAudio
}

func splitChatFiles(text string) (string, []chatFile) {
	i := strings.LastIndex(text, chatFileBegin)
	if i < 0 || !strings.HasSuffix(text, chatFileEnd) {
		return text, nil
	}
	body := text[:i]
	raw := strings.TrimSuffix(text[i+len(chatFileBegin):], chatFileEnd)
	var files []chatFile
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var f chatFile
		if json.Unmarshal([]byte(line), &f) != nil || strings.TrimSpace(f.Path) == "" {
			return text, nil
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		return text, nil
	}
	return body, files
}

func joinChatFiles(body string, files []chatFile) string {
	files = mergeChatFiles(nil, files)
	if len(files) == 0 {
		return body
	}
	var b strings.Builder
	b.WriteString(body)
	b.WriteString(chatFileBegin)
	for _, f := range files {
		line, err := json.Marshal(f)
		if err != nil {
			continue
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	b.WriteString(strings.TrimPrefix(chatFileEnd, "\n"))
	return b.String()
}

func mergeChatFiles(base, extra []chatFile) []chatFile {
	out := append([]chatFile{}, base...)
	seen := map[string]bool{}
	for _, f := range out {
		seen[f.Path] = true
	}
	for _, f := range extra {
		f.Path = strings.TrimSpace(f.Path)
		if f.Path == "" || seen[f.Path] {
			continue
		}
		if f.Name == "" {
			f.Name = filepath.Base(f.Path)
		}
		seen[f.Path] = true
		out = append(out, f)
	}
	return out
}

func (a *Agent) bindChatFiles(text string) string {
	if a == nil {
		return text
	}
	return withChatFileRefs(a.Cfg.WorkspaceRoot, a.Cfg.WorkspaceID, text)
}

func withChatFileRefs(root, workspaceID, text string) string {
	body, files := splitChatFiles(text)
	found := findChatFileRefs(root, body, workspaceAgentNames(workspaceID))
	return joinChatFiles(body, mergeChatFiles(files, found))
}

func resolveChatFileRefs(root, text string, agentNames []string) string {
	body, files := splitChatFiles(text)
	found := findChatFileRefs(root, body, agentNames)
	return joinChatFiles(body, mergeChatFiles(files, found))
}

func workspaceAgentNames(workspaceID string) []string {
	if strings.TrimSpace(workspaceID) == "" {
		return nil
	}
	agents, err := listAgentInstancesForWorkspace(workspaceID)
	if err != nil {
		return nil
	}
	var names []string
	for _, m := range agents {
		if name := strings.TrimSpace(m.Name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func findChatFileRefs(root, text string, agentNames []string) []chatFile {
	var out []chatFile
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		if runes[i] != '@' {
			continue
		}
		if i > 0 && !unicode.IsSpace(runes[i-1]) {
			continue
		}
		token, n := chatFileToken(runes[i+1:])
		if token == "" {
			continue
		}
		if agentNameToken(token, agentNames) {
			continue
		}
		path, ok := resolveChatFilePath(root, token)
		if !ok {
			trimmed := strings.TrimRight(token, ".,;:!?")
			if trimmed != token {
				path, ok = resolveChatFilePath(root, trimmed)
			}
		}
		if !ok {
			continue
		}
		source := "path"
		if insideWorkspace(root, path) {
			source = "workspace"
		}
		out = append(out, chatFile{Name: filepath.Base(path), Path: path, Source: source})
		i += n
	}
	return out
}

func chatFileToken(rest []rune) (string, int) {
	if len(rest) == 0 {
		return "", 0
	}
	if rest[0] == '"' {
		for i := 1; i < len(rest); i++ {
			if rest[i] == '"' {
				return string(rest[1:i]), i + 1
			}
		}
		return "", 0
	}
	i := 0
	for i < len(rest) && !unicode.IsSpace(rest[i]) {
		i++
	}
	return string(rest[:i]), i
}

func agentNameToken(token string, names []string) bool {
	for _, name := range names {
		if name != "" && strings.EqualFold(token, name) {
			return true
		}
	}
	return false
}

func resolveChatFilePath(root, token string) (string, bool) {
	token = strings.TrimSpace(token)
	if token == "" || strings.Contains(token, "://") {
		return "", false
	}
	token = filepath.Clean(filepath.FromSlash(token))
	var path string
	if filepath.IsAbs(token) {
		path = token
	} else {
		if strings.TrimSpace(root) == "" {
			return "", false
		}
		path = filepath.Join(root, token)
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return "", false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path, true
	}
	return abs, true
}

func insideWorkspace(root, path string) bool {
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// copyDroppedChatFile stores a local file the user dragged in. Dolphin
// gives the window a file:// URL rather than the file bytes.
func copyDroppedChatFile(root, raw string) (chatFile, error) {
	path, err := localDropPath(raw)
	if err != nil {
		return chatFile{}, err
	}
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() {
		return chatFile{}, fmt.Errorf("not a file")
	}
	if st.Size() > chatUploadLimit {
		return chatFile{}, fmt.Errorf("file is larger than %dMB", chatUploadLimit>>20)
	}
	f, err := os.Open(path)
	if err != nil {
		return chatFile{}, err
	}
	defer f.Close()
	return saveChatUpload(root, filepath.Base(path), f)
}

func localDropPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("not a local file")
	}
	if strings.HasPrefix(strings.ToLower(raw), "file:") {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "file" {
			return "", fmt.Errorf("not a local file")
		}
		host := u.Hostname()
		if host != "" && !strings.EqualFold(host, "localhost") {
			return "", fmt.Errorf("not a local file")
		}
		p, err := url.PathUnescape(u.Path)
		if err != nil || p == "" {
			return "", fmt.Errorf("not a local file")
		}
		if len(p) >= 3 && p[0] == '/' && p[2] == ':' {
			p = p[1:]
		}
		raw = p
	}
	drive := len(raw) >= 3 && raw[1] == ':'
	if !filepath.IsAbs(raw) && !drive {
		return "", fmt.Errorf("not a local file")
	}
	return filepath.Clean(raw), nil
}

func saveChatUpload(root, name string, r io.Reader) (chatFile, error) {
	name = safeUploadName(name)
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return chatFile{}, err
	}
	id := hex.EncodeToString(buf[:])
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return chatFile{}, err
	}
	path := filepath.Join(dir, name)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return chatFile{}, err
	}
	defer f.Close()
	n, err := io.Copy(f, io.LimitReader(r, chatUploadLimit+1))
	if err != nil {
		return chatFile{}, err
	}
	if n > chatUploadLimit {
		_ = os.RemoveAll(dir)
		return chatFile{}, fmt.Errorf("file is larger than %dMB", chatUploadLimit>>20)
	}
	return chatFile{ID: id, Name: name, Path: path, Source: "temp"}, nil
}

func openChatUpload(root, id string) (chatFile, error) {
	id = strings.TrimSpace(id)
	if len(id) != 16 || strings.Trim(id, "0123456789abcdef") != "" {
		return chatFile{}, fmt.Errorf("unknown attachment")
	}
	dir := filepath.Join(root, id)
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 || entries[0].IsDir() {
		return chatFile{}, fmt.Errorf("unknown attachment")
	}
	name := entries[0].Name()
	return chatFile{ID: id, Name: name, Path: filepath.Join(dir, name), Source: "temp"}, nil
}

func safeUploadName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, strings.TrimSpace(name))
	if name == "" || name == "." || name == ".." {
		return "file"
	}
	if len(name) > 180 {
		name = name[len(name)-180:]
	}
	return name
}

func chatFileNames(files []chatFile) []string {
	if len(files) == 0 {
		return nil
	}
	names := make([]string, 0, len(files))
	for _, f := range files {
		name := strings.TrimSpace(f.Name)
		if name == "" {
			name = filepath.Base(f.Path)
		}
		names = append(names, name)
	}
	return names
}

// expandChatFiles turns attached-file trailers into model input.
// Text is included. Audio bytes are included only when the model can hear.
// An image stays as its path. The bytes go out only on a delegate call.
func expandChatFiles(items []json.RawMessage, images, audio bool) []json.RawMessage {
	out := make([]json.RawMessage, len(items))
	for i, raw := range items {
		out[i] = expandChatFileItem(raw, images, audio)
	}
	return out
}

func expandChatFileItem(raw json.RawMessage, images, audio bool) json.RawMessage {
	var item map[string]any
	if json.Unmarshal(raw, &item) != nil {
		return raw
	}
	role, _ := item["role"].(string)
	content, ok := item["content"].(string)
	if !strings.EqualFold(role, "user") || !ok {
		return raw
	}
	body, files := splitChatFiles(content)
	if len(files) == 0 {
		return raw
	}
	text, parts := chatFileModelContent(body, files, images, audio)
	if len(parts) == 0 {
		item["content"] = text
	} else {
		item["content"] = append([]any{map[string]any{"type": "input_text", "text": text}}, parts...)
	}
	next, err := json.Marshal(item)
	if err != nil {
		return raw
	}
	return next
}

func chatFileModelContent(body string, files []chatFile, images, audio bool) (string, []any) {
	var b strings.Builder
	b.WriteString(strings.TrimRight(body, "\n"))
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	b.WriteString("Attached files. The path is the file. Short text is included below. Use tools on the path for anything not included. Copy a file into the workspace only when the user asks.\n")
	var parts []any
	for _, f := range files {
		b.WriteString("- ")
		b.WriteString(f.Name)
		b.WriteString(" (")
		b.WriteString(f.Source)
		b.WriteString(") ")
		b.WriteString(f.Path)
		b.WriteByte('\n')
		if chatFileKind(f.Path) == "image" {
			if images {
				b.WriteString("  ")
				b.WriteString(imageDelegateHint)
				b.WriteByte('\n')
			}
			continue
		}
		kind, data := readChatFile(f.Path)
		switch kind {
		case "text":
			b.WriteString("<file path=\"")
			b.WriteString(f.Path)
			b.WriteString("\">\n")
			b.Write(data)
			if len(data) == 0 || data[len(data)-1] != '\n' {
				b.WriteByte('\n')
			}
			b.WriteString("</file>\n")
		case "audio":
			format := audioFormat(f.Path)
			if audio && format != "" && len(data) > 0 && len(data) <= chatAudioLimit {
				parts = append(parts, map[string]any{
					"type": "input_audio",
					"input_audio": map[string]any{
						"data":   base64.StdEncoding.EncodeToString(data),
						"format": format,
					},
				})
			}
		}
	}
	return b.String(), parts
}

func readChatFile(path string) (string, []byte) {
	f, err := os.Open(path)
	if err != nil {
		return "missing", nil
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		return "missing", nil
	}
	switch chatFileKind(path) {
	case "image":
		if st.Size() > chatImageLimit {
			return "other", nil
		}
		b, err := io.ReadAll(io.LimitReader(f, chatImageLimit+1))
		if err != nil || len(b) > chatImageLimit {
			return "other", nil
		}
		return "image", b
	case "audio":
		if st.Size() > chatAudioLimit {
			return "other", nil
		}
		b, err := io.ReadAll(io.LimitReader(f, chatAudioLimit+1))
		if err != nil || len(b) > chatAudioLimit {
			return "other", nil
		}
		return "audio", b
	case "text":
		if st.Size() > chatInlineLimit {
			return "other", nil
		}
		b, err := io.ReadAll(io.LimitReader(f, chatInlineLimit+1))
		if err != nil || len(b) > chatInlineLimit || !utf8.Valid(b) || strings.ContainsRune(string(b), 0) {
			return "other", nil
		}
		return "text", b
	default:
		return "other", nil
	}
}

func chatFileKind(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return "image"
	case ".wav", ".mp3", ".m4a", ".ogg", ".flac", ".aac":
		return "audio"
	case ".txt", ".md", ".markdown", ".html", ".htm", ".css", ".js", ".mjs", ".ts", ".json", ".xml", ".csv", ".tsv", ".yml", ".yaml", ".go", ".py", ".rs", ".toml", ".sh", ".sql", ".svg":
		return "text"
	default:
		return "other"
	}
}

func imageMIME(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/png"
	}
}

func audioFormat(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".wav":
		return "wav"
	case ".mp3":
		return "mp3"
	default:
		return ""
	}
}

func searchWorkspaceFiles(root, query string, limit int) []map[string]string {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" || strings.TrimSpace(root) == "" || limit <= 0 {
		return nil
	}
	var out []map[string]string
	visited := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		visited++
		if visited > 5000 || len(out) >= limit {
			return filepath.SkipAll
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "dist") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if strings.Contains(strings.ToLower(rel), query) {
			out = append(out, map[string]string{"name": filepath.Base(path), "path": rel})
		}
		return nil
	})
	return out
}

func catalogModalities(input []string, modality string) (images, audio bool) {
	check := func(s string) {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "image", "vision":
			images = true
		case "audio":
			audio = true
		}
	}
	for _, s := range input {
		check(s)
	}
	if i := strings.Index(modality, "->"); i >= 0 {
		modality = modality[:i]
	}
	for _, part := range strings.FieldsFunc(strings.ToLower(modality), func(r rune) bool {
		return r == '+' || r == ',' || r == '/' || r == ' '
	}) {
		check(part)
	}
	return images, audio
}

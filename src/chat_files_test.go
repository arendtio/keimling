package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseModelCatalogReadsModalities(t *testing.T) {
	raw := []byte(`{"data":[
		{"id":"see","architecture":{"modality":"text+image->text","input_modalities":["text","image"]}},
		{"id":"hear","input_modalities":["text","audio"]},
		{"id":"both","architecture":{"modality":"text+image+audio->text"}},
		{"id":"quiet","name":"Quiet"}
	]}`)
	got, err := parseModelCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]catalogModel{}
	for _, m := range got {
		byID[m.ID] = m
	}
	if !byID["see"].SeesImages || byID["see"].HearsAudio {
		t.Fatalf("see: %#v", byID["see"])
	}
	if byID["hear"].SeesImages || !byID["hear"].HearsAudio {
		t.Fatalf("hear: %#v", byID["hear"])
	}
	if !byID["both"].SeesImages || !byID["both"].HearsAudio {
		t.Fatalf("both: %#v", byID["both"])
	}
	if byID["quiet"].SeesImages || byID["quiet"].HearsAudio {
		t.Fatalf("quiet: %#v", byID["quiet"])
	}
}

func TestNativeSensesDefaultOnAndCanBeDisabled(t *testing.T) {
	on := providerModel{SeesImages: true, HearsAudio: true}
	if !on.nativeImages() || !on.nativeAudio() {
		t.Fatalf("default: images %v audio %v", on.nativeImages(), on.nativeAudio())
	}
	off := false
	disabled := providerModel{ID: "m", SeesImages: true, HearsAudio: true, UseImages: &off, UseAudio: &off}
	if disabled.nativeImages() || disabled.nativeAudio() {
		t.Fatal("explicit off should stay off")
	}
	unknown := providerModel{UseImages: boolPtr(true), UseAudio: boolPtr(true)}
	if unknown.nativeImages() || unknown.nativeAudio() {
		t.Fatal("no detected capability should not turn the sense on")
	}
	out := publicModelProviders([]modelProvider{{
		ID: "p", Name: "P", BaseURL: "http://127.0.0.1:9/v1", ActiveModelID: "m",
		Models: []providerModel{disabled},
	}})
	models := out[0]["models"].([]map[string]any)
	if models[0]["sees_images"] != true || models[0]["use_images"] != false {
		t.Fatalf("public model: %#v", models[0])
	}
}

func TestAtFileRefKeepsWorkspaceFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "NOTES.md")
	if err := os.WriteFile(path, []byte("Saturday market"), 0o644); err != nil {
		t.Fatal(err)
	}
	text := withChatFileRefs(root, "", "read @NOTES.md please")
	body, files := splitChatFiles(text)
	if body != "read @NOTES.md please" {
		t.Fatalf("body %q", body)
	}
	if len(files) != 1 || files[0].Path != path || files[0].Source != "workspace" {
		t.Fatalf("%#v", files)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 1 {
		t.Fatalf("workspace gained a copy: %d", len(entries))
	}
}

func TestAtFileRefOutsideWorkspaceIsNotCopied(t *testing.T) {
	ws := t.TempDir()
	outside := t.TempDir()
	path := filepath.Join(outside, "shot.png")
	if err := os.WriteFile(path, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	text := withChatFileRefs(ws, "", "see @"+path)
	_, files := splitChatFiles(text)
	if len(files) != 1 || files[0].Path != path || files[0].Source != "path" {
		t.Fatalf("%#v", files)
	}
	entries, _ := os.ReadDir(ws)
	if len(entries) != 0 {
		t.Fatalf("workspace changed: %d", len(entries))
	}
}

func TestAtFileSkipsAgentNameEmailAndMissing(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "NOTES.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	text := resolveChatFileRefs(root, "mail a@NOTES.md @Missing.md @NOTES.md", []string{"NOTES.md"})
	if strings.Contains(text, "keimling_files") {
		t.Fatalf("agent name and missing path should not attach: %s", text)
	}
	text = resolveChatFileRefs(root, "mail a@NOTES.md @Missing.md", nil)
	if strings.Contains(text, "keimling_files") {
		t.Fatalf("email and missing path attached: %s", text)
	}
}

func TestDroppedFileURLCopiesLikeAnUpload(t *testing.T) {
	srcDir := t.TempDir()
	store := t.TempDir()
	src := filepath.Join(srcDir, "mpv-shot0001.jpg")
	if err := os.WriteFile(src, []byte("jpeg-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := copyDroppedChatFile(store, "file://"+src)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "temp" || got.Name != "mpv-shot0001.jpg" {
		t.Fatalf("%#v", got)
	}
	body, err := os.ReadFile(got.Path)
	if err != nil || string(body) != "jpeg-bytes" {
		t.Fatalf("%q %v", body, err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatal(err)
	}
	if rel, err := filepath.Rel(srcDir, got.Path); err == nil && !strings.HasPrefix(rel, "..") {
		t.Fatalf("copy landed beside the original: %s", got.Path)
	}

	spaced := filepath.Join(srcDir, "my shot.jpg")
	if err := os.WriteFile(spaced, []byte("spaced"), 0o644); err != nil {
		t.Fatal(err)
	}
	escaped := strings.ReplaceAll(spaced, " ", "%20")
	got, err = copyDroppedChatFile(store, "file://"+escaped)
	if err != nil || got.Name != "my shot.jpg" {
		t.Fatalf("%#v %v", got, err)
	}
	if _, err := copyDroppedChatFile(store, "file://files.example/tmp/x.jpg"); err == nil {
		t.Fatal("remote file url should fail")
	}
	if _, err := copyDroppedChatFile(store, "relative.jpg"); err == nil {
		t.Fatal("relative path should fail")
	}
	if _, err := copyDroppedChatFile(store, "file://"+srcDir); err == nil {
		t.Fatal("directory should fail")
	}
}

func TestDroppedFileStaysOutsideWorkspace(t *testing.T) {
	ws := t.TempDir()
	store := t.TempDir()
	f, err := saveChatUpload(store, "scan.pdf", strings.NewReader("%PDF"))
	if err != nil {
		t.Fatal(err)
	}
	if f.Source != "temp" || f.Name != "scan.pdf" {
		t.Fatalf("%#v", f)
	}
	if rel, err := filepath.Rel(ws, f.Path); err == nil && !strings.HasPrefix(rel, "..") {
		t.Fatalf("upload landed in the workspace: %s", f.Path)
	}
	got, err := openChatUpload(store, f.ID)
	if err != nil || got.Path != f.Path {
		t.Fatalf("%#v %v", got, err)
	}
	if _, err := openChatUpload(store, "../"+f.ID); err == nil {
		t.Fatal("id escape should fail")
	}
}

func TestExpandChatFilesInlinesTextAndGatesSenses(t *testing.T) {
	dir := t.TempDir()
	note := filepath.Join(dir, "note.md")
	img := filepath.Join(dir, "pic.png")
	wav := filepath.Join(dir, "hi.wav")
	if err := os.WriteFile(note, []byte("# hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(img, []byte{0x89, 0x50, 0x4e, 0x47}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wav, []byte("RIFFwave"), 0o644); err != nil {
		t.Fatal(err)
	}
	files := []chatFile{
		{Name: "note.md", Path: note, Source: "workspace"},
		{Name: "pic.png", Path: img, Source: "temp"},
		{Name: "hi.wav", Path: wav, Source: "temp"},
	}
	raw, _ := json.Marshal(map[string]any{"role": "user", "content": joinChatFiles("look", files)})
	off := expandChatFiles([]json.RawMessage{raw}, false, false)
	var item map[string]any
	if err := json.Unmarshal(off[0], &item); err != nil {
		t.Fatal(err)
	}
	text, _ := item["content"].(string)
	if !strings.Contains(text, "# hello") || !strings.Contains(text, note) || !strings.Contains(text, img) {
		t.Fatalf("text and paths: %s", text)
	}
	if strings.Contains(text, "input_image") || strings.Contains(text, "base64") {
		t.Fatalf("senses off should not embed media: %s", text)
	}
	on := expandChatFiles([]json.RawMessage{raw}, true, true)
	if err := json.Unmarshal(on[0], &item); err != nil {
		t.Fatal(err)
	}
	parts, ok := item["content"].([]any)
	if !ok || len(parts) < 2 {
		t.Fatalf("content %#v", item["content"])
	}
	blob, _ := json.Marshal(parts)
	s := string(blob)
	if strings.Contains(s, "input_image") || strings.Contains(s, "iVBORw") {
		t.Fatalf("image bytes belong on the delegate call: %s", s)
	}
	if !strings.Contains(s, "input_audio") || !strings.Contains(s, "# hello") || !strings.Contains(s, imageDelegateHint) {
		t.Fatalf("native parts: %s", s)
	}
	if !strings.Contains(s, img) {
		t.Fatal("path missing while seeing is on")
	}
	again := expandChatFiles([]json.RawMessage{raw}, true, true)
	if !strings.Contains(string(again[0]), img) || strings.Contains(string(again[0]), "input_image") {
		t.Fatal("a later turn lost the path or gained the bytes")
	}
}

func TestSessionMessageListsFilesWithoutTrailer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "NOTES.md")
	raw, _ := json.Marshal(map[string]any{
		"role":    "user",
		"content": joinChatFiles("read this", []chatFile{{Name: "NOTES.md", Path: path, Source: "workspace"}}),
	})
	msgs := sessionMessages([]json.RawMessage{raw}, 10)
	if len(msgs) != 1 || msgs[0].Text != "read this" || len(msgs[0].Files) != 1 || msgs[0].Files[0] != "NOTES.md" {
		t.Fatalf("%#v", msgs)
	}
	if strings.Contains(msgs[0].Text, "keimling_files") {
		t.Fatal(msgs[0].Text)
	}
}

func TestChatTargetUsesSavedSenses(t *testing.T) {
	off := false
	p := &modelProvider{ID: "p", BaseURL: "http://127.0.0.1:9/v1"}
	m := &providerModel{ID: "google/gemini-4-argon", SeesImages: true, HearsAudio: true, UseAudio: &off}
	got := chatTargetFrom(p, m, 8000, "high")
	if !got.SeeImages || got.HearAudio {
		t.Fatalf("see %v hear %v", got.SeeImages, got.HearAudio)
	}
	plain := chatTargetFrom(p, &providerModel{ID: "text-only"}, 8000, "high")
	if plain.SeeImages || plain.HearAudio {
		t.Fatal("a model without a detected sense must not receive the file as media")
	}
}

func boolPtr(v bool) *bool { return &v }

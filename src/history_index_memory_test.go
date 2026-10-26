package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildHistoryIndexDoesNotUseUserMessageAsTitle(t *testing.T) {
	events := []json.RawMessage{
		json.RawMessage(`{"role":"user","content":"Investigate PTY deadlock in detail please"}`),
		json.RawMessage(`{"type":"function_call","name":"exec","call_id":"1","arguments":"{}"}`),
		json.RawMessage(`{"role":"user","content":"Now add retries"}`),
	}
	idx := buildHistoryIndex(events, HistoryIndex{Version: historyIndexVersion})
	var tasks []HistorySection
	for _, s := range idx.Sections {
		if s.Level == "task" {
			tasks = append(tasks, s)
		}
	}
	if len(tasks) != 2 {
		t.Fatalf("tasks: %d", len(tasks))
	}
	if tasks[0].Title == "Investigate PTY deadlock in detail please" {
		t.Fatalf("title must not be the user message: %q", tasks[0].Title)
	}
	if !strings.HasPrefix(tasks[0].Title, "Chapter ") {
		t.Fatalf("provisional title expected, got %q", tasks[0].Title)
	}
	if tasks[0].Memory != memoryRaw {
		t.Fatalf("memory: %q", tasks[0].Memory)
	}
}

func TestApplyMemoryFromCompaction(t *testing.T) {
	idx := HistoryIndex{
		Version: historyIndexVersion,
		Sections: []HistorySection{
			{ID: "task-1", Level: "task", Title: "Chapter 1", Start: 1, End: 10, Memory: memoryRaw},
			{ID: "task-2", Level: "task", Title: "Chapter 2", Start: 11, End: 20, Memory: memoryRaw},
		},
	}
	summary := `<semantic_history>
[section depth=1 1-10: Old work | summarize]
done
[section depth=1 11-20: Live | distill]
busy
</semantic_history>`
	applyMemoryFromCompaction(&idx, summary)
	if idx.Sections[0].Memory != memorySummarize {
		t.Fatalf("task1 memory: %q", idx.Sections[0].Memory)
	}
	if idx.Sections[1].Memory != memoryDistill {
		t.Fatalf("task2 memory: %q", idx.Sections[1].Memory)
	}
}

func TestForCompactionSkipsSystemAndDeveloper(t *testing.T) {
	in := []json.RawMessage{
		json.RawMessage(`{"role":"system","content":"You are keimling…"}`),
		json.RawMessage(`{"role":"developer","content":"hidden"}`),
		json.RawMessage(`{"role":"user","content":"hi"}`),
	}
	out := forCompaction(in)
	if len(out) != 1 || !strings.Contains(string(out[0]), `"user"`) {
		t.Fatalf("got %#v", out)
	}
}

func TestCleanGeneratedTitle(t *testing.T) {
	if got := cleanGeneratedTitle("\"PTY deadlock fix\"\nmore"); got != "PTY deadlock fix" {
		t.Fatalf("got %q", got)
	}
}

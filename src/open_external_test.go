package main

import "testing"

func TestOpenExternalURLRejectsBadSchemes(t *testing.T) {
	for _, raw := range []string{"", "file:///etc/passwd", "javascript:alert(1)", "not a url"} {
		if err := openExternalURL(raw); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}

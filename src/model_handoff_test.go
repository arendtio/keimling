package main

import "testing"

func TestSlashCompletion(t *testing.T) {
	cases := []struct {
		in      string
		pos     int
		want    string
		matches int
	}{
		{"/stat", 5, "/status", 1},
		{"/continue-n", 11, "/continue-n", 0},
		{"/con", 4, "/continue", 1},
		{"/mode y", 7, "/mode y", 0},
		{"hi /sta", 7, "hi /sta", 0},
	}
	for _, tc := range cases {
		got, _, matches := completeSlash(tc.in, tc.pos)
		if got != tc.want || len(matches) != tc.matches {
			t.Errorf("%q got %q (%v), want %q (%d)", tc.in, got, matches, tc.want, tc.matches)
		}
	}
}

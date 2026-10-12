package main

import (
	"strings"
	"testing"
)

func TestSplitMessageRespectsRuneLimitAndPreservesText(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxLength int
	}{
		{name: "single long line", text: strings.Repeat("x", 17), maxLength: 5},
		{name: "prefers newline", text: "one\ntwo\nthree", maxLength: 7},
		{name: "unicode", text: strings.Repeat("😀", 9), maxLength: 4},
		{name: "prefers spaces", text: "alpha beta gamma", maxLength: 7},
		{name: "already fits", text: "short", maxLength: 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			chunks := splitMessage(tc.text, tc.maxLength)
			if len(chunks) == 0 {
				t.Fatal("splitMessage returned no chunks")
			}
			if got := strings.Join(chunks, ""); got != tc.text {
				t.Fatalf("chunks changed the message: got %q, want %q", got, tc.text)
			}
			for i, chunk := range chunks {
				if got := len([]rune(chunk)); got > tc.maxLength {
					t.Errorf("chunk %d has %d runes; limit is %d", i, got, tc.maxLength)
				}
			}
		})
	}
}

func TestSplitMessageNonPositiveLimitReturnsOriginal(t *testing.T) {
	for _, limit := range []int{0, -1} {
		got := splitMessage("unchanged", limit)
		if len(got) != 1 || got[0] != "unchanged" {
			t.Fatalf("splitMessage(%d) = %#v, want original text", limit, got)
		}
	}
}

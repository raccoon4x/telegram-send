package telegram

import (
	"strings"
	"testing"
)

func TestSendFilesRejectsUnsupportedType(t *testing.T) {
	c, err := NewClient("", "", "token", "chat", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.SendFiles([]string{"a.jpg"}, "caption", "Video"); err == nil {
		t.Fatal("expected error for unsupported file type")
	}
}

func TestSplitText(t *testing.T) {
	text := strings.Repeat("a", 3000) + "\n" + strings.Repeat("b", 3000)
	got := splitText(text, MaxMessageLength)
	if len(got) != 2 || got[0] != strings.Repeat("a", 3000) || got[1] != strings.Repeat("b", 3000) {
		t.Fatalf("expected split at line break, got %d chunks", len(got))
	}

	// Emoji outside the BMP take two UTF-16 code units each.
	got = splitText(strings.Repeat("😀", 3000), MaxMessageLength)
	if len(got) != 2 || TextLength(got[0]) != 4096 || TextLength(got[1]) != 1904 {
		t.Fatalf("unexpected emoji split: %d chunks", len(got))
	}
}

package editor

import "testing"

func TestListPrefix(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantPrefix string
		wantEnd    int
		wantContent bool
	}{
		{"unordered dash with content", "- hello", "- ", 2, true},
		{"unordered star with content", "* item", "* ", 2, true},
		{"unordered plus with content", "+ item", "+ ", 2, true},
		{"empty dash marker", "- ", "- ", 2, false},
		{"ordered with content", "1. first", "2. ", 3, true},
		{"ordered 9 with content", "9. ninth", "10. ", 3, true},
		{"ordered paren", "3) item", "4) ", 3, true},
		{"empty ordered marker", "1. ", "2. ", 3, false},
		{"indented bullet", "  - nested", "  - ", 4, true},
		{"not a list", "hello world", "", 0, false},
		{"blank line", "", "", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, end, hasContent := listPrefix([]rune(tt.line))
			if prefix != tt.wantPrefix {
				t.Errorf("prefix = %q, want %q", prefix, tt.wantPrefix)
			}
			if end != tt.wantEnd {
				t.Errorf("markerEnd = %d, want %d", end, tt.wantEnd)
			}
			if hasContent != tt.wantContent {
				t.Errorf("hasContent = %v, want %v", hasContent, tt.wantContent)
			}
		})
	}
}

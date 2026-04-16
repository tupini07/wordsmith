package editor

import (
	"testing"
)

func TestHighlightHeading(t *testing.T) {
	theme := GruvboxTheme()
	line := []rune("# Hello World")
	tokens, inFM := HighlightLine(line, theme, false, 5)

	if inFM {
		t.Error("heading should not set frontmatter state")
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token for heading, got %d", len(tokens))
	}
	if tokens[0].Text != "# Hello World" {
		t.Errorf("expected full heading text, got %q", tokens[0].Text)
	}
}

func TestHighlightFrontmatter(t *testing.T) {
	theme := GruvboxTheme()

	// Opening delimiter (must be line 0)
	_, inFM := HighlightLine([]rune("---"), theme, false, 0)
	if !inFM {
		t.Error("--- on line 0 should enter frontmatter")
	}

	// Content inside frontmatter
	tokens, inFM := HighlightLine([]rune("title: test"), theme, true, 1)
	if !inFM {
		t.Error("should still be in frontmatter")
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}

	// Closing delimiter
	_, inFM = HighlightLine([]rune("---"), theme, true, 2)
	if inFM {
		t.Error("closing --- should exit frontmatter")
	}
}

func TestHighlightHRNotFrontmatter(t *testing.T) {
	theme := GruvboxTheme()

	// --- on a non-zero line when not in frontmatter → HR, not frontmatter
	_, inFM := HighlightLine([]rune("---"), theme, false, 10)
	if inFM {
		t.Error("--- on a non-first line should NOT enter frontmatter")
	}
}

func TestHighlightBlockquote(t *testing.T) {
	theme := GruvboxTheme()
	tokens, _ := HighlightLine([]rune("> quoted text"), theme, false, 5)
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token for blockquote, got %d", len(tokens))
	}
}

func TestHighlightInlineBold(t *testing.T) {
	theme := GruvboxTheme()
	tokens, _ := HighlightLine([]rune("hello **bold** world"), theme, false, 5)

	// Should have: "hello ", "**bold**", " world"
	if len(tokens) < 3 {
		t.Fatalf("expected at least 3 tokens, got %d", len(tokens))
	}

	foundBold := false
	for _, tok := range tokens {
		if tok.Text == "**bold**" {
			foundBold = true
		}
	}
	if !foundBold {
		t.Error("expected a bold token")
	}
}

func TestHighlightInlineItalic(t *testing.T) {
	theme := GruvboxTheme()
	tokens, _ := HighlightLine([]rune("hello *italic* world"), theme, false, 5)

	foundItalic := false
	for _, tok := range tokens {
		if tok.Text == "*italic*" {
			foundItalic = true
		}
	}
	if !foundItalic {
		t.Error("expected an italic token")
	}
}

func TestHighlightInlineCode(t *testing.T) {
	theme := GruvboxTheme()
	tokens, _ := HighlightLine([]rune("use `fmt.Println` here"), theme, false, 5)

	foundCode := false
	for _, tok := range tokens {
		if tok.Text == "`fmt.Println`" {
			foundCode = true
		}
	}
	if !foundCode {
		t.Error("expected an inline code token")
	}
}

func TestHighlightLink(t *testing.T) {
	theme := GruvboxTheme()
	tokens, _ := HighlightLine([]rune("click [here](https://example.com) now"), theme, false, 5)

	foundLink := false
	foundURL := false
	for _, tok := range tokens {
		if tok.Text == "[here]" {
			foundLink = true
		}
		if tok.Text == "(https://example.com)" {
			foundURL = true
		}
	}
	if !foundLink {
		t.Error("expected a link text token")
	}
	if !foundURL {
		t.Error("expected a link URL token")
	}
}

func TestHighlightListMarker(t *testing.T) {
	theme := GruvboxTheme()
	tokens, _ := HighlightLine([]rune("- list item"), theme, false, 5)

	if len(tokens) < 2 {
		t.Fatalf("expected at least 2 tokens, got %d", len(tokens))
	}
	if tokens[0].Text != "- " {
		t.Errorf("expected list marker %q, got %q", "- ", tokens[0].Text)
	}
}

func TestHighlightNumberedList(t *testing.T) {
	theme := GruvboxTheme()
	tokens, _ := HighlightLine([]rune("1. first item"), theme, false, 5)

	if len(tokens) < 2 {
		t.Fatalf("expected at least 2 tokens, got %d", len(tokens))
	}
	if tokens[0].Text != "1. " {
		t.Errorf("expected list marker %q, got %q", "1. ", tokens[0].Text)
	}
}

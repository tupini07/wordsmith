package editor

import (
	"testing"
)

var noState = HighlightState{}

func TestHighlightHeading(t *testing.T) {
	theme := GruvboxTheme()
	line := []rune("# Hello World")
	tokens, st := HighlightLine(line, theme, noState, 5)

	if st.InFrontmatter || st.InCodeBlock {
		t.Error("heading should not change state")
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
	_, st := HighlightLine([]rune("---"), theme, noState, 0)
	if !st.InFrontmatter {
		t.Error("--- on line 0 should enter frontmatter")
	}

	// Content inside frontmatter
	tokens, st := HighlightLine([]rune("title: test"), theme, HighlightState{InFrontmatter: true}, 1)
	if !st.InFrontmatter {
		t.Error("should still be in frontmatter")
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}

	// Closing delimiter
	_, st = HighlightLine([]rune("---"), theme, HighlightState{InFrontmatter: true}, 2)
	if st.InFrontmatter {
		t.Error("closing --- should exit frontmatter")
	}
}

func TestHighlightHRNotFrontmatter(t *testing.T) {
	theme := GruvboxTheme()

	// --- on a non-zero line when not in frontmatter → HR, not frontmatter
	_, st := HighlightLine([]rune("---"), theme, noState, 10)
	if st.InFrontmatter {
		t.Error("--- on a non-first line should NOT enter frontmatter")
	}
}

func TestHighlightCodeFence(t *testing.T) {
	theme := GruvboxTheme()

	// Opening fence
	_, st := HighlightLine([]rune("```sql"), theme, noState, 5)
	if !st.InCodeBlock {
		t.Error("opening ``` should enter code block")
	}

	// Content inside code block
	tokens, st := HighlightLine([]rune("SELECT * FROM users;"), theme, HighlightState{InCodeBlock: true}, 6)
	if !st.InCodeBlock {
		t.Error("should still be in code block")
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token for code block content, got %d", len(tokens))
	}

	// Closing fence
	_, st = HighlightLine([]rune("```"), theme, HighlightState{InCodeBlock: true}, 7)
	if st.InCodeBlock {
		t.Error("closing ``` should exit code block")
	}
}

func TestHighlightBlockquote(t *testing.T) {
	theme := GruvboxTheme()
	tokens, state := HighlightLine([]rune("> quoted text"), theme, noState, 5)
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token for blockquote, got %d", len(tokens))
	}
	if !state.InBlockquote {
		t.Fatal("expected InBlockquote to be true after blockquote line")
	}

	// Continuation sub-line should also be styled as blockquote
	contTokens, _ := HighlightLine([]rune("continuation of quote"), theme, state, 5)
	if len(contTokens) != 1 {
		t.Fatalf("expected 1 token for blockquote continuation, got %d", len(contTokens))
	}
}

func TestHighlightInlineBold(t *testing.T) {
	theme := GruvboxTheme()
	tokens, _ := HighlightLine([]rune("hello **bold** world"), theme, noState, 5)

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
	tokens, _ := HighlightLine([]rune("hello *italic* world"), theme, noState, 5)

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

func TestHighlightRawURL(t *testing.T) {
	theme := GruvboxTheme()

	// URL in the middle of text
	tokens, _ := HighlightLine([]rune("visit https://example.com/path for info"), theme, noState, 5)
	foundURL := false
	for _, tok := range tokens {
		if tok.Text == "https://example.com/path" {
			foundURL = true
		}
	}
	if !foundURL {
		t.Errorf("expected URL token, got tokens: %v", tokTexts(tokens))
	}

	// URL at end of sentence (trailing period stripped)
	tokens2, _ := HighlightLine([]rune("see http://example.com."), theme, noState, 5)
	foundURL2 := false
	for _, tok := range tokens2 {
		if tok.Text == "http://example.com" {
			foundURL2 = true
		}
	}
	if !foundURL2 {
		t.Errorf("expected URL without trailing period, got tokens: %v", tokTexts(tokens2))
	}
}

func tokTexts(tokens []Token) []string {
	var out []string
	for _, t := range tokens {
		out = append(out, t.Text)
	}
	return out
}

func TestHighlightInlineCode(t *testing.T) {
	theme := GruvboxTheme()
	tokens, _ := HighlightLine([]rune("use `fmt.Println` here"), theme, noState, 5)

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
	tokens, _ := HighlightLine([]rune("click [here](https://example.com) now"), theme, noState, 5)

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
	tokens, _ := HighlightLine([]rune("- list item"), theme, noState, 5)

	if len(tokens) < 2 {
		t.Fatalf("expected at least 2 tokens, got %d", len(tokens))
	}
	if tokens[0].Text != "- " {
		t.Errorf("expected list marker %q, got %q", "- ", tokens[0].Text)
	}
}

func TestHighlightNumberedList(t *testing.T) {
	theme := GruvboxTheme()
	tokens, _ := HighlightLine([]rune("1. first item"), theme, noState, 5)

	if len(tokens) < 2 {
		t.Fatalf("expected at least 2 tokens, got %d", len(tokens))
	}
	if tokens[0].Text != "1. " {
		t.Errorf("expected list marker %q, got %q", "1. ", tokens[0].Text)
	}
}

func TestSliceTokens(t *testing.T) {
	theme := GruvboxTheme()
	// "hello *italic* world" — tokens: ["hello ", "*italic*", " world"]
	tokens, _ := HighlightLine([]rune("hello *italic* world"), theme, noState, 5)

	// Slice the middle — should preserve italic style across the cut
	sliced := SliceTokens(tokens, 4, 12)
	combined := ""
	for _, tok := range sliced {
		combined += tok.Text
	}
	if combined != "o *itali" {
		t.Errorf("expected %q, got %q", "o *itali", combined)
	}

	// Check that sliced tokens preserve styles (italic portion should be styled)
	if len(sliced) < 2 {
		t.Fatalf("expected at least 2 tokens in slice, got %d", len(sliced))
	}
}

func TestSliceTokensFullLine(t *testing.T) {
	theme := GruvboxTheme()
	line := []rune("hello **bold** world")
	tokens, _ := HighlightLine(line, theme, noState, 5)

	// Slicing the full range should give identical text
	sliced := SliceTokens(tokens, 0, len(line))
	combined := ""
	for _, tok := range sliced {
		combined += tok.Text
	}
	if combined != string(line) {
		t.Errorf("full slice = %q, want %q", combined, string(line))
	}
}

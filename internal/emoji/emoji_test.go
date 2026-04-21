package emoji

import "testing"

func TestSearchEmpty(t *testing.T) {
	results := Search("", 5)
	if len(results) != 5 {
		t.Errorf("expected 5 results for empty query, got %d", len(results))
	}
}

func TestSearchByName(t *testing.T) {
	results := Search("fire", 10)
	found := false
	for _, e := range results {
		if e.Emoji == "🔥" {
			found = true
		}
	}
	if !found {
		t.Error("expected to find fire emoji 🔥")
	}
}

func TestSearchNoMatch(t *testing.T) {
	results := Search("xyznonexistent", 10)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	results := Search("HEART", 10)
	if len(results) == 0 {
		t.Error("expected case-insensitive search to find results")
	}
}

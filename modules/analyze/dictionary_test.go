package analyze

import (
	"testing"
)

func TestLoadDictionaryFromJSON(t *testing.T) {
	jsonData := []byte(`{"positive_emotion": ["happy", "joy"], "negative_emotion": ["sad", "angry"]}`)
	dict, err := LoadDictionaryFromJSON(jsonData)
	if err != nil {
		t.Fatalf("LoadDictionaryFromJSON error: %v", err)
	}

	cats := dict.Categories()
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(cats))
	}

	if got := dict.Lookup("happy"); len(got) != 1 || got[0] != "positive_emotion" {
		t.Errorf("Lookup(happy) = %v; want [positive_emotion]", got)
	}
	if got := dict.Lookup("sad"); len(got) != 1 || got[0] != "negative_emotion" {
		t.Errorf("Lookup(sad) = %v; want [negative_emotion]", got)
	}
	if got := dict.Lookup("unknown"); len(got) != 0 {
		t.Errorf("Lookup(unknown) = %v; want []", got)
	}
}

func TestDictionaryCaseInsensitive(t *testing.T) {
	jsonData := []byte(`{"test_cat": ["Hello"]}`)
	dict, err := LoadDictionaryFromJSON(jsonData)
	if err != nil {
		t.Fatalf("LoadDictionaryFromJSON error: %v", err)
	}
	if got := dict.Lookup("hello"); len(got) != 1 {
		t.Errorf("Lookup(hello) = %v; want 1 category", got)
	}
	if got := dict.Lookup("HELLO"); len(got) != 1 {
		t.Errorf("Lookup(HELLO) = %v; want 1 category", got)
	}
}

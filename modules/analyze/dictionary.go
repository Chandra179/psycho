package analyze

import (
	"encoding/json"
	"strings"
)

// Category represents a psycholinguistic category label.
type Category string

// Dictionary is the lookup interface for psycholinguistic categories.
type Dictionary interface {
	Lookup(word string) []Category
	Categories() []Category
}

// builtInDictionary implements Dictionary using a word-to-categories map.
type builtInDictionary struct {
	wordToCats map[string][]Category
	cats       []Category
}

// LoadDictionaryFromJSON parses a JSON dictionary file.
// Expected format: {"category_name": ["word1", "word2", ...], ...}
func LoadDictionaryFromJSON(data []byte) (Dictionary, error) {
	var raw map[string][]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	wordToCats := make(map[string][]Category)
	var cats []Category
	for catName, words := range raw {
		cat := Category(catName)
		cats = append(cats, cat)
		for _, w := range words {
			w = strings.ToLower(strings.TrimSpace(w))
			if w == "" {
				continue
			}
			wordToCats[w] = append(wordToCats[w], cat)
		}
	}
	return &builtInDictionary{wordToCats: wordToCats, cats: cats}, nil
}

func (d *builtInDictionary) Lookup(word string) []Category {
	word = strings.ToLower(word)
	if cats, ok := d.wordToCats[word]; ok {
		return cats
	}
	return nil
}

func (d *builtInDictionary) Categories() []Category {
	return d.cats
}

package analyze

import "strings"

// negationWindow is how many tokens before a word may carry a negator that
// flips its meaning ("did not care", "not happy"). Three covers "never really
// cared" and "not at all happy" without reaching into the previous clause.
const negationWindow = 3

var negators = map[string]bool{
	"not": true, "no": true, "never": true, "nothing": true, "nobody": true,
	"neither": true, "nor": true, "cannot": true, "without": true, "hardly": true,
	"barely": true, "t": true,
}

// isNegator reports whether a lowercase token negates what follows. Contracted
// negations ("didn't", "isn't", "won't") end in n't; "t" is what remains of
// "didn’t" after the tokenizer splits at a curly apostrophe.
func isNegator(w string) bool {
	return negators[w] || strings.HasSuffix(w, "n't") || strings.HasSuffix(w, "n’t")
}

// negatedAt reports whether any of the negationWindow tokens before index i is
// a negator. Features and excerpts both call it so the highlighted words and
// the counts cannot disagree.
func negatedAt(words []string, i int) bool {
	for j := i - 1; j >= 0 && j >= i-negationWindow; j-- {
		if isNegator(words[j]) {
			return true
		}
	}
	return false
}

// isValueCategory reports whether cat is a Schwartz value list.
func isValueCategory(cat Category) bool {
	return strings.HasPrefix(string(cat), "value_")
}

// countedCategories returns the categories that word i contributes to. Value
// words are dropped when negated, so rejecting a value ("did not care") does not
// count as endorsing it. Other categories keep their LIWC-style plain counts
// because the trait weights were derived from plain counts.
func countedCategories(words []string, i int, dict Dictionary) []Category {
	cats := dict.Lookup(words[i])
	if len(cats) == 0 {
		return nil
	}
	var negated, checked bool
	out := cats[:0:0]
	for _, c := range cats {
		if isValueCategory(c) {
			if !checked {
				negated, checked = negatedAt(words, i), true
			}
			if negated {
				continue
			}
		}
		out = append(out, c)
	}
	return out
}

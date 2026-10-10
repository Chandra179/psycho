package ingest

import (
	"errors"
	"strings"
)

// ErrNotEnglish reports text that does not look like English. The dictionary,
// the trait weights and the reference corpus are all English, so scores for
// another language would look real and mean nothing.
var ErrNotEnglish = errors.New("This tool reads English text only. The text does not look like English, so no scores were calculated.")

// Below minEnglishCheckWords there is too little text for the share to be
// stable, so the check passes and the quality flag handles short input.
const minEnglishCheckWords = 30

// minEnglishShare is the least share of words that must be common English
// function words. Ordinary English prose runs about 30 to 45 percent; other
// languages mostly fall under 10.
const minEnglishShare = 0.18

var englishFunctionWords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`the be to of and a in that have i it for not on with he as you do at this but his by from they we say her she or an will my one all would there their what so up out if about who get which go me when make can like time no just him know take people into year your good some could them see other than then now look only come its over think also back after use two how our work first well way even new want because any these give day most us is are was were been has had did`) {
		englishFunctionWords[w] = true
	}
}

// CheckEnglish returns ErrNotEnglish when the text has enough words and too few
// of them are common English function words. It is a rough guard against
// meaningless scores, not a language identifier: a few short or technical
// texts may pass or fail by chance.
func CheckEnglish(doc Document) error {
	words := tokenizeWords(doc.RawText)
	if len(words) < minEnglishCheckWords {
		return nil
	}
	hits := 0
	for _, w := range words {
		if englishFunctionWords[strings.ToLower(w)] {
			hits++
		}
	}
	if float64(hits)/float64(len(words)) < minEnglishShare {
		return ErrNotEnglish
	}
	return nil
}

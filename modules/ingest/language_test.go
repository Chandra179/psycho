package ingest

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckEnglish(t *testing.T) {
	english := "Yesterday I walked through the old market with my sister and we talked about our plans for the future. I feel happy and curious about everything we might learn, because new ideas excite me. We discussed the books and the traditions that shape what we value."
	other := map[string]string{
		"spanish":    "Ayer caminé por el mercado con mi hermana y hablamos de nuestros planes para el futuro. Me siento feliz y curioso por todo lo que podemos aprender, porque las ideas nuevas me emocionan mucho. Hablamos de los libros, de la familia y de las tradiciones que nos hacen quienes somos.",
		"german":     "Gestern bin ich mit meiner Schwester über den alten Markt gegangen und wir haben über unsere Pläne für die Zukunft gesprochen. Ich fühle mich glücklich und neugierig auf alles, was wir lernen können, weil neue Ideen mich begeistern. Wir sprachen über Bücher und die Familie.",
		"indonesian": "Kemarin saya berjalan di pasar tua bersama kakak perempuan saya dan kami membicarakan rencana kami untuk masa depan. Saya merasa senang dan penasaran dengan semua yang bisa kami pelajari, karena ide-ide baru membuat saya bersemangat. Kami membahas buku dan keluarga.",
		"portuguese": "Ontem caminhei pelo mercado antigo com a minha irmã e conversamos sobre os nossos planos para o futuro. Sinto-me feliz e curioso com tudo o que podemos aprender, porque as ideias novas me entusiasmam. Falamos sobre livros, sobre a família e sobre as tradições.",
	}
	norm := NewNormalizer()
	if err := CheckEnglish(norm.Normalize(english)); err != nil {
		t.Fatalf("English rejected: %v", err)
	}
	for name, text := range other {
		if err := CheckEnglish(norm.Normalize(text)); !errors.Is(err, ErrNotEnglish) {
			t.Errorf("%s accepted: %v", name, err)
		}
	}
	// Too little text to judge: passes and leaves it to the quality flag.
	if err := CheckEnglish(norm.Normalize("Hola, como estas hoy amigo mio")); err != nil {
		t.Errorf("short text rejected: %v", err)
	}
	if strings.ContainsRune(ErrNotEnglish.Error(), '—') {
		t.Error("error message must not use em-dashes")
	}
}

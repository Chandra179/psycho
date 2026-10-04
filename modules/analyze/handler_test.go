package analyze

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"psycho/modules/ingest"
	"psycho/zlogger"
)

func TestURLInvalidTextReturns400(t *testing.T) {
	for _, text := range []string{"", "   \n\t\u2003", "<p> </p>", "!!!!!!!!!!!!"} {
		reached := false
		pipeline := func(ctx context.Context, sourceType, sourceDate, raw string) (ingest.AnalysisOutput, error) {
			reached = true
			return ingest.AnalysisOutput{}, ingest.ValidateDocument(ingest.NewNormalizer().Normalize(raw))
		}
		fetch := func(url string, limit int) (string, error) { return text, nil }
		h := makeHandleAnalyze(100000, zlogger.New("prod"), pipeline, fetch)
		body, _ := json.Marshal(AnalyzeRequest{SourceType: "url", SourceURL: "https://example.com/writing"})
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest("POST", "/analyze", strings.NewReader(string(body))))
		if !reached || w.Code != 400 {
			t.Fatalf("invalid fetched text: pipeline=%v status=%d", reached, w.Code)
		}
	}
}

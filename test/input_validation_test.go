package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"psycho/modules/analyze"
	"psycho/modules/ingest"
	"psycho/modules/report"
	"psycho/zlogger"
)

func TestInvalidTextHTTPPaths(t *testing.T) {
	pipe, _ := newTestPipeline(t)
	logger := zlogger.New("prod")
	emptyDir := t.TempDir()
	w := httptest.NewRecorder()
	ingest.MakeHandleAnalyzeDir(ingest.Config{DirPath: emptyDir, MaxTextSize: 100000}, logger, pipe.Run)(w, httptest.NewRequest("POST", "/analyze-dir", strings.NewReader("{}")))
	if w.Code != 400 {
		t.Fatalf("empty directory status: %d", w.Code)
	}
	for _, text := range []string{"", "               ", "\t\n\r\u2003\u00a0", "!!!!!!!!!!!!!!", "<p></p><div> </div>", "123456789"} {
		payload, _ := json.Marshal(map[string]string{"source_type": "paste", "text": text})
		w := httptest.NewRecorder()
		analyze.MakeHandleAnalyze(100000, logger, pipe.Run)(w, httptest.NewRequest("POST", "/analyze", strings.NewReader(string(payload))))
		if w.Code != 400 {
			t.Errorf("JSON accepted invalid text %q: %d", text, w.Code)
		}
		for _, hx := range []bool{true, false} {
			form := url.Values{"text": {text}, "consent": {"on"}}
			r := httptest.NewRequest("POST", "/report", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if hx {
				r.Header.Set("HX-Request", "true")
			}
			w := httptest.NewRecorder()
			report.MakeHandleReportForm(100000, "../templates", logger, pipe.Run)(w, r)
			if w.Code != 400 {
				t.Errorf("browser accepted invalid text %q: %d", text, w.Code)
			}
		}
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "text.txt"), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		w = httptest.NewRecorder()
		ingest.MakeHandleAnalyzeDir(ingest.Config{DirPath: dir, MaxTextSize: 100000}, logger, pipe.Run)(w, httptest.NewRequest("POST", "/analyze-dir", strings.NewReader("{}")))
		if w.Code != 400 {
			t.Errorf("directory accepted invalid text %q: %d: %s", text, w.Code, w.Body.String())
		}
	}
}

func TestValidInputDetailsInBothAPIs(t *testing.T) {
	pipe, pdeps := newTestPipeline(t)
	logger := zlogger.New("prod")
	text := "This is valid writing about the research and plans. I feel happy with friends."
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "text.txt"), []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	handlers := []http.HandlerFunc{analyze.MakeHandleAnalyze(100000, logger, pipe.Run), ingest.MakeHandleAnalyzeDir(ingest.Config{DirPath: dir, MaxTextSize: 100000}, logger, pipe.Run)}
	for _, h := range handlers {
		payload, _ := json.Marshal(map[string]string{"text": text, "source_type": "paste"})
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest("POST", "/analyze", strings.NewReader(string(payload))))
		if w.Code != 200 {
			t.Fatalf("valid input: %d: %s", w.Code, w.Body.String())
		}
		var out struct {
			AnalysisID string                      `json:"analysis_id"`
			Details    *analyze.CalculationDetails `json:"calculation_details"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if out.Details == nil || len(out.Details.Traits) != 9 {
			t.Fatal("API lost calculation details")
		}
		saved, err := pdeps.Storage.GetAnalysis(out.AnalysisID)
		if err != nil {
			t.Fatal(err)
		}
		if saved.CalculationDetails == nil {
			t.Fatal("retrieval lost calculation details")
		}
	}
}

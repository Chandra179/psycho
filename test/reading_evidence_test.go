package integration_test

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"psycho/modules/analyze"
	"psycho/modules/ingest"
	"psycho/modules/profile"
	"psycho/zlogger"
)

func TestContextualEvidenceAcrossHTTPResponseShapes(t *testing.T) {
	pipe, deps := newTestPipeline(t)
	logger := zlogger.New("prod")
	text := "Our cultural heritage influences culture. We question tradition and authority."
	data, _ := json.Marshal(map[string]string{"text": text})
	rec := httptest.NewRecorder()
	analyze.MakeHandleAnalyze(100000, logger, pipe.Run)(rec, httptest.NewRequest("POST", "/analyze", strings.NewReader(string(data))))
	if rec.Code != 200 {
		t.Fatalf("analyze: %d %s", rec.Code, rec.Body.String())
	}
	var analysis analyze.AnalyzeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &analysis); err != nil {
		t.Fatal(err)
	}
	if len(analysis.ValueExcerpts["value_tradition"]) == 0 {
		t.Fatal("analyze response dropped excerpts")
	}

	request := httptest.NewRequest("GET", "/analysis/"+analysis.AnalysisID, nil)
	request.SetPathValue("id", analysis.AnalysisID)
	rec = httptest.NewRecorder()
	profile.MakeHandleGetAnalysis(deps.Storage, logger)(rec, request)
	var saved profile.SavedAnalysis
	if rec.Code != 200 {
		t.Fatalf("retrieval: %d %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved.ValueExcerpts, analysis.ValueExcerpts) {
		t.Fatal("retrieval changed contextual evidence")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "writing.txt"), []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	ingest.MakeHandleAnalyzeDir(ingest.Config{DirPath: dir, MaxTextSize: 100000}, logger, pipe.Run)(rec, httptest.NewRequest("POST", "/analyze-dir", strings.NewReader("{}")))
	if rec.Code != 200 {
		t.Fatalf("directory: %d %s", rec.Code, rec.Body.String())
	}
	var batch ingest.AnalyzeDirResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &batch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(batch.ValueExcerpts, analysis.ValueExcerpts) || len(batch.Traits) != 9 {
		t.Fatal("directory response lost evidence or measures")
	}
}

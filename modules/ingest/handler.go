package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"psycho/middleware"
	"psycho/zlogger"
)

type AnalyzeDirRequest struct {
	SourceType string `json:"source_type" validate:"omitempty,oneof=blog chat email paste file url"`
	SourceDate string `json:"source_date" validate:"omitempty,datetime=2006-01-02"`
}

type AnalyzeDirResponse struct {
	AnalysisID         string             `json:"analysis_id"`
	WordCount          int                `json:"word_count"`
	DictionaryCoverage float64            `json:"dictionary_coverage"`
	ConfidenceFlag     string             `json:"confidence_flag"`
	Traits             map[string]any     `json:"traits"`
	Values             map[string]float64 `json:"values"`
	FilesRead          int                `json:"files_read"`
	Summary            any                `json:"summary"`
	Narrative          string             `json:"narrative"`
}

// AnalysisOutput is everything the HTTP layer needs to render a response
// after a successful analysis. Field types stay loose because ingest cannot
// import its sibling modules without an import cycle; the JSON shape is the
// contract at this seam.
type AnalysisOutput struct {
	AnalysisID         string
	WordCount          int
	DictionaryCoverage float64
	ConfidenceFlag     string
	Traits             map[string]any
	Values             map[string]float64
	Summary            any
	Narrative          string
}

// AnalyzeFunc is the seam the HTTP handlers call into. modules/server and
// the tests wire it to a *pipeline.Pipeline.
type AnalyzeFunc func(ctx context.Context, sourceType, sourceDate, text string) (AnalysisOutput, error)

func MakeHandleAnalyzeDir(
	cfg Config,
	logger *zlogger.Logger,
	analyzeFn AnalyzeFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := middleware.DecodeAndValidate[AnalyzeDirRequest](r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if cfg.DirPath == "" {
			http.Error(w, "dir_path not configured", http.StatusServiceUnavailable)
			return
		}

		sourceType := req.SourceType
		if sourceType == "" {
			sourceType = "file"
		}

		text, filesRead, err := ReadDir(cfg.DirPath, cfg.MaxTextSize)
		if err != nil {
			logger.Error(r.Context(), "failed to read directory", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "failed to read directory: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if len(text) < 10 {
			http.Error(w, "combined text must be at least 10 characters", http.StatusBadRequest)
			return
		}

		if cfg.MaxTextSize > 0 && len(text) > cfg.MaxTextSize {
			http.Error(w, "combined text exceeds max size", http.StatusBadRequest)
			return
		}

		out, err := analyzeFn(r.Context(), sourceType, req.SourceDate, text)
		if err != nil {
			logger.Error(r.Context(), "analysis failed", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := AnalyzeDirResponse{
			AnalysisID:         out.AnalysisID,
			WordCount:          out.WordCount,
			DictionaryCoverage: out.DictionaryCoverage,
			ConfidenceFlag:     out.ConfidenceFlag,
			Traits:             out.Traits,
			Values:             out.Values,
			FilesRead:          filesRead,
			Summary:            out.Summary,
			Narrative:          out.Narrative,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// urlFetchClient bounds every fetch in time and refuses redirects to
// loopback, private, or link-local addresses, so a public URL cannot be
// used to probe the local network or cloud metadata endpoints.
var urlFetchClient = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		if err := checkURLHost(req.URL); err != nil {
			return fmt.Errorf("redirect to disallowed address: %w", err)
		}
		return nil
	},
}

// FetchURLText downloads a URL and returns its body as text, capped at
// maxSize bytes (no cap when maxSize <= 0). The scheme must be http or
// https, and hosts resolving to private addresses are refused — this runs
// on a user-supplied URL, so it must never become a free proxy into the
// local network, and the body must never be read unbounded into memory.
func FetchURLText(rawURL string, maxSize int) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	if err := checkURLHost(u); err != nil {
		return "", err
	}

	resp, err := urlFetchClient.Get(rawURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" &&
		!strings.HasPrefix(ct, "text/") &&
		!strings.Contains(ct, "html") &&
		!strings.Contains(ct, "json") &&
		!strings.Contains(ct, "xml") {
		return "", fmt.Errorf("unsupported content type %q", ct)
	}

	var body io.Reader = resp.Body
	if maxSize > 0 {
		body = io.LimitReader(resp.Body, int64(maxSize)+1)
	}
	b, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	if maxSize > 0 && len(b) > maxSize {
		return "", fmt.Errorf("response exceeds max size %d", maxSize)
	}
	return string(b), nil
}

// checkURLHost rejects non-http(s) schemes and any host that is, or
// resolves to, a private address.
func checkURLHost(u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("missing host")
	}
	if ip := net.ParseIP(host); ip != nil {
		return rejectPrivateIP(ip)
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve host: %w", err)
	}
	for _, ip := range ips {
		if err := rejectPrivateIP(ip); err != nil {
			return err
		}
	}
	return nil
}

func rejectPrivateIP(ip net.IP) error {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return fmt.Errorf("refusing to fetch private address %s", ip)
	}
	return nil
}

// ReadDir concatenates the .txt files in dirPath, stopping once the combined
// text exceeds maxSize (no cap when maxSize <= 0) so a huge directory cannot
// exhaust memory before the caller's size check rejects the request.
func ReadDir(dirPath string, maxSize int) (string, int, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", 0, fmt.Errorf("read dir %s: %w", dirPath, err)
	}

	var builder strings.Builder
	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".txt") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dirPath, e.Name()))
		if err != nil {
			return "", 0, fmt.Errorf("read file %s: %w", e.Name(), err)
		}
		if count > 0 {
			builder.WriteString("\n\n")
		}
		builder.Write(b)
		count++
		if maxSize > 0 && builder.Len() > maxSize {
			break
		}
	}

	if count == 0 {
		return "", 0, fmt.Errorf("no .txt files found in %s", dirPath)
	}

	return builder.String(), count, nil
}

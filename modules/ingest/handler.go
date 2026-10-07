package ingest

import (
	"context"
	"encoding/json"
	"errors"
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

type AnalyzeDirRequest struct{}

type AnalyzeDirResponse struct {
	AnalysisID          string                   `json:"analysis_id"`
	WordCount           int                      `json:"word_count"`
	DictionaryCoverage  float64                  `json:"dictionary_coverage"`
	ConfidenceFlag      string                   `json:"confidence_flag"`
	Traits              map[string]any           `json:"traits"`
	Values              map[string]float64       `json:"values"`
	ValueEvidence       map[string][]string      `json:"value_evidence,omitempty"`
	ValueExcerpts       map[string][]TextExcerpt `json:"value_excerpts,omitempty"`
	PercentileReference *PercentileReference     `json:"percentile_reference,omitempty"`
	CalculationDetails  any                      `json:"calculation_details,omitempty"`
	FilesRead           int                      `json:"files_read"`
	Summary             any                      `json:"summary"`
	Narrative           string                   `json:"narrative"`
}

const (
	PercentileMethodEmpirical           = "empirical"
	PercentileMethodNormalApproximation = "normal_approximation"
)

// PercentileReference describes how analysis percentiles were produced.
type PercentileReference struct {
	Method     string `json:"method"`
	Corpus     string `json:"corpus,omitempty"`
	SampleSize int    `json:"sample_size,omitempty"`
}

// AnalysisOutput is everything the HTTP layer needs to render a response
// after a successful analysis. Field types stay loose because ingest cannot
// import its sibling modules without an import cycle; the JSON shape is the
// contract at this seam, so the tags below are load-bearing — consumers
// (including the report renderer) decode this shape by its snake_case keys.
type AnalysisOutput struct {
	AnalysisID          string                   `json:"analysis_id"`
	WordCount           int                      `json:"word_count"`
	DictionaryCoverage  float64                  `json:"dictionary_coverage"`
	ConfidenceFlag      string                   `json:"confidence_flag"`
	Traits              map[string]any           `json:"traits"`
	Values              map[string]float64       `json:"values"`
	ValueEvidence       map[string][]string      `json:"value_evidence,omitempty"`
	ValueExcerpts       map[string][]TextExcerpt `json:"value_excerpts,omitempty"`
	PercentileReference *PercentileReference     `json:"percentile_reference,omitempty"`
	CalculationDetails  any                      `json:"calculation_details,omitempty"`
	Summary             any                      `json:"summary"`
	Narrative           string                   `json:"narrative,omitempty"`
}

// AnalyzeFunc is the seam the HTTP handlers call into. modules/server and
// the tests wire it to a *pipeline.Pipeline.
type AnalyzeFunc func(ctx context.Context, text string) (AnalysisOutput, error)

func MakeHandleAnalyzeDir(
	cfg Config,
	logger *zlogger.Logger,
	analyzeFn AnalyzeFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10) // the dir request is tiny JSON; refuse anything bigger
		_, err := middleware.DecodeAndValidate[AnalyzeDirRequest](r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if cfg.DirPath == "" {
			http.Error(w, "dir_path not configured", http.StatusServiceUnavailable)
			return
		}

		text, filesRead, err := ReadDir(cfg.DirPath, cfg.MaxTextSize)
		if err != nil {
			if errors.Is(err, ErrInvalidText) {
				http.Error(w, ErrInvalidText.Error(), http.StatusBadRequest)
				return
			}
			logger.Error(r.Context(), "failed to read directory", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "failed to read directory: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if cfg.MaxTextSize > 0 && len(text) > cfg.MaxTextSize {
			http.Error(w, "combined text exceeds max size", http.StatusBadRequest)
			return
		}

		out, err := analyzeFn(r.Context(), text)
		if err != nil {
			if errors.Is(err, ErrInvalidText) {
				http.Error(w, ErrInvalidText.Error(), http.StatusBadRequest)
				return
			}
			logger.Error(r.Context(), "analysis failed", zlogger.Field{Key: "error", Value: err.Error()})
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := AnalyzeDirResponse{
			AnalysisID:          out.AnalysisID,
			WordCount:           out.WordCount,
			DictionaryCoverage:  out.DictionaryCoverage,
			ConfidenceFlag:      out.ConfidenceFlag,
			Traits:              out.Traits,
			Values:              out.Values,
			ValueEvidence:       out.ValueEvidence,
			ValueExcerpts:       out.ValueExcerpts,
			PercentileReference: out.PercentileReference,
			CalculationDetails:  out.CalculationDetails,
			FilesRead:           filesRead,
			Summary:             out.Summary,
			Narrative:           out.Narrative,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// urlFetchClient bounds every fetch in time and refuses redirects to
// loopback, private, or link-local addresses, so a public URL cannot be
// used to probe the local network or cloud metadata endpoints. The Dial
// hook re-validates the resolved IP at connection time: pre-fetch DNS
// checks alone are TOCTOU-vulnerable to rebinding answers that flip to a
// private address between validation and dial.
var urlFetchClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		DialContext: dialCheckedAddr,
	},
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

// dialCheckedAddr resolves addr's host itself, rejects any private address,
// and dials the validated IP. TLS still uses the URL's hostname for SNI and
// certificate verification, so this cannot be abused to bypass certs.
func dialCheckedAddr(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("split dial address: %w", err)
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve host: %w", err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("host %q resolved to no addresses", host)
	}
	for _, ip := range ips {
		if err := rejectPrivateIP(ip.IP); err != nil {
			return nil, err
		}
	}
	var dialer net.Dialer
	return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
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
		if count > 0 {
			builder.WriteString("\n\n")
		}
		file, err := os.Open(filepath.Join(dirPath, e.Name()))
		if err != nil {
			return "", 0, fmt.Errorf("read file %s: %w", e.Name(), err)
		}
		var reader io.Reader = file
		if maxSize > 0 {
			reader = io.LimitReader(file, max(0, int64(maxSize)-int64(builder.Len())+1))
		}
		_, readErr := io.Copy(&builder, reader)
		closeErr := file.Close()
		if readErr != nil {
			return "", 0, fmt.Errorf("read file %s: %w", e.Name(), readErr)
		}
		if closeErr != nil {
			return "", 0, fmt.Errorf("close file %s: %w", e.Name(), closeErr)
		}
		count++
		if maxSize > 0 && builder.Len() > maxSize {
			break
		}
	}

	if count == 0 {
		return "", 0, fmt.Errorf("no .txt files found in %s: %w", dirPath, ErrInvalidText)
	}

	return builder.String(), count, nil
}

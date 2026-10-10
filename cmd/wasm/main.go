//go:build js && wasm

// Command wasm is the browser build of Psycho. It compiles the same analysis,
// scoring and report code the server uses to WebAssembly, so a visitor's text
// is analyzed on their own device and never leaves it. The page script calls
// the functions this file registers on globalThis.psycho.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"syscall/js"

	"psycho/assets"
	"psycho/config"
	"psycho/modules/analyze"
	"psycho/modules/ingest"
	"psycho/modules/pipeline"
	"psycho/modules/profile"
	"psycho/modules/report"
	"psycho/templates"
)

// maxTextBytes caps pasted text at about 1 MiB.
const maxTextBytes = 1 << 20

func main() {
	deps, err := analyze.NewDependenciesFromData(analyze.EmbeddedDictionary, config.EmbeddedCalibration)
	if err != nil {
		js.Global().Set("psycho", map[string]any{"error": err.Error()})
		select {}
	}

	agg := profile.NewScoreAggregator()
	agg.UseCalibration(deps.Calibration)
	// Nothing is stored here; the page keeps history on the visitor's device.
	pipe := pipeline.New(deps.Extractor, deps.Model, agg, profile.NewTemplateNarrativeGenerator(), deps.Calibration)

	js.Global().Set("psycho", map[string]any{
		"ready":   true,
		"analyze": js.FuncOf(func(_ js.Value, args []js.Value) any { return analyzeText(pipe, args) }),
		"render":  js.FuncOf(func(_ js.Value, args []js.Value) any { return renderJSON(args) }),
	})

	// Keep the Go runtime alive so the registered callbacks stay valid.
	select {}
}

// analyzeText runs the pipeline on args[0] and returns
// {ok, json, html} or {ok:false, error}.
func analyzeText(pipe *pipeline.Pipeline, args []js.Value) any {
	if len(args) < 1 {
		return failure("no text given")
	}
	text := args[0].String()
	if len(text) > maxTextBytes {
		return failure("Text exceeds the maximum size.")
	}
	out, err := pipe.Run(context.Background(), text)
	if err != nil {
		if errors.Is(err, ingest.ErrInvalidText) {
			return failure(ingest.ErrInvalidText.Error())
		}
		return failure("Something went wrong while analyzing; please try again.")
	}
	blob, err := json.Marshal(out)
	if err != nil {
		return failure("Something went wrong while analyzing; please try again.")
	}
	html, err := renderBlob(blob, false)
	if err != nil {
		return failure("Something went wrong while building the report.")
	}
	return map[string]any{"ok": true, "json": string(blob), "html": html}
}

// renderJSON renders a saved analysis (args[0], the JSON analyze returned)
// again. args[1] true asks for a complete standalone page with the stylesheet
// embedded, for the "Save report" download; otherwise it returns the fragment.
func renderJSON(args []js.Value) any {
	if len(args) < 1 {
		return failure("no analysis given")
	}
	standalone := len(args) > 1 && args[1].Truthy()
	html, err := renderBlob([]byte(args[0].String()), standalone)
	if err != nil {
		return failure("This saved reading could not be shown.")
	}
	return map[string]any{"ok": true, "html": html}
}

func renderBlob(blob []byte, standalone bool) (string, error) {
	var a report.Analysis
	if err := json.Unmarshal(blob, &a); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if standalone {
		err := report.RenderStandaloneAnalysisFS(templates.FS, assets.CSS, &a, &buf)
		return buf.String(), err
	}
	err := report.RenderAnalysisFS(templates.FS, &a, &buf, false)
	return buf.String(), err
}

func failure(msg string) map[string]any {
	return map[string]any{"ok": false, "error": msg}
}

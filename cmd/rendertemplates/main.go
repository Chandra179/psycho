// One-off script: renders the single HTML report (templates/report.html)
// against an analysis JSON (as returned by POST /analyze or /analyze-dir,
// read from stdin) and writes profile-report.html to the current directory.
// The view-building logic lives in modules/report, which also powers the
// server's POST /report response.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"psycho/modules/report"
)

func main() {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	var a report.Analysis
	if err := json.Unmarshal(raw, &a); err != nil {
		panic(err)
	}

	f, err := os.Create("profile-report.html")
	if err != nil {
		panic(err)
	}
	if err := report.RenderStandaloneAnalysis("templates", &a, f); err != nil {
		f.Close()
		panic(err)
	}
	f.Close()

	fmt.Println("rendered profile-report.html")
}

// Package templates holds the HTML templates compiled into the WebAssembly
// build, which cannot read files at run time.
package templates

import "embed"

//go:embed report.html report-page.html
var FS embed.FS

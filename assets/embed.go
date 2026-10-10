// Package assets holds the compiled stylesheet used by the WebAssembly build's
// standalone report download.
package assets

import _ "embed"

//go:embed app.css
var CSS []byte

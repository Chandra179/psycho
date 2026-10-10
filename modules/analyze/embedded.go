package analyze

import _ "embed"

// EmbeddedDictionary is the committed dictionary, compiled into binaries
// that cannot read files at run time (the WebAssembly build).
//
//go:embed dictionary.json
var EmbeddedDictionary []byte

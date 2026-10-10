package config

import _ "embed"

// EmbeddedCalibration is the committed reference calibration, compiled into
// the WebAssembly build.
//
//go:embed calibration.json
var EmbeddedCalibration []byte

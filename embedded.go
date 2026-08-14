package embedded

import _ "embed"

// ToolsJSON is the default manifest bundled into the executable.
//
//go:embed tools.json
var ToolsJSON []byte

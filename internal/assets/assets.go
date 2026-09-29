// Package assets embeds static resources (logo, etc.) so they ship inside the
// compiled binary and don't depend on the runtime working directory.
package assets

import _ "embed"

// LogoPNG is the XinText logo embedded at build time. Used by HTML export
// to render a self-contained brand bar (base64 data URL).
//
//go:embed logo.png
var LogoPNG []byte

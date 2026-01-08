package templates

import "embed"

//go:embed *.html partials/*.html
var FS embed.FS

package static

import "embed"

//go:embed css/* js/*
var FS embed.FS

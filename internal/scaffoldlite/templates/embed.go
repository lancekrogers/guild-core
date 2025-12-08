package templates

import (
	"embed"
	"io/fs"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// EmbeddedTemplates contains the embedded template files
//
//go:embed templates
var EmbeddedTemplates embed.FS

// GetEmbeddedTemplatesFS returns the embedded templates filesystem
func GetEmbeddedTemplatesFS() (fs.FS, error) {
	subFS, err := fs.Sub(EmbeddedTemplates, "templates")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get templates subdirectory")
	}
	return subFS, nil
}

// GetTemplatesFS returns the full embedded filesystem including the templates directory
func GetTemplatesFS() fs.FS {
	return EmbeddedTemplates
}

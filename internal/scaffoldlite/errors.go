package scaffoldlite

import (
	"fmt"
	"strings"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

const (
	ErrCodeValidation        = gerror.ErrCodeValidation
	ErrCodeTemplateNotFound  = gerror.ErrCodeNotFound
	ErrCodeTemplateRender    = gerror.ErrCodeInternal
	ErrCodeFileExists        = gerror.ErrCodeAlreadyExists
	ErrCodeFileWrite         = gerror.ErrCodeIO
	ErrCodeFileRead          = gerror.ErrCodeIO
	ErrCodeYAMLParse         = gerror.ErrCodeParsing
	ErrCodeSchemaValidation  = gerror.ErrCodeValidation
	ErrCodeTimeout           = gerror.ErrCodeTimeout
	ErrCodePermission        = gerror.ErrCodePermissionDenied
	ErrCodeInvalidPath       = gerror.ErrCodeInvalidInput
	ErrCodeRecipeNotFound    = gerror.ErrCodeNotFound
	ErrCodeVariableUndefined = gerror.ErrCodeInvalidInput
)

type ValidationError struct {
	Field   string           `json:"field"`
	Message string           `json:"message"`
	Value   any              `json:"value,omitempty"`
	Code    gerror.ErrorCode `json:"code"`
	Line    int              `json:"line,omitempty"`
	Column  int              `json:"column,omitempty"`
}

func (ve ValidationError) Error() string {
	location := ""
	if ve.Line > 0 {
		if ve.Column > 0 {
			location = fmt.Sprintf(" (line %d, col %d)", ve.Line, ve.Column)
		} else {
			location = fmt.Sprintf(" (line %d)", ve.Line)
		}
	}
	if ve.Value != nil {
		return fmt.Sprintf("%s: %s (value: %v)%s", ve.Field, ve.Message, ve.Value, location)
	}
	return fmt.Sprintf("%s: %s%s", ve.Field, ve.Message, location)
}

type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "validation failed with no specific errors"
	}
	if len(ve) == 1 {
		return ve[0].Error()
	}
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("validation failed with %d errors:\n", len(ve)))
	for i, err := range ve {
		buf.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return buf.String()
}

func ErrRecipeNotFound(path string) error {
	return gerror.New(ErrCodeRecipeNotFound, "recipe file not found", nil).WithDetails("path", path)
}

func ErrTemplateNotFound(template, templatesDir string) error {
	return gerror.New(ErrCodeTemplateNotFound, "template file not found", nil).
		WithDetails("template", template).WithDetails("templatesDir", templatesDir)
}

func ErrFileExists(path string) error {
	return gerror.New(ErrCodeFileExists, "file already exists and overwrite is disabled", nil).WithDetails("path", path)
}

func ErrTemplateRender(template string, err error) error {
	return gerror.Wrap(err, ErrCodeTemplateRender, "template rendering failed").WithDetails("template", template)
}

func ErrFileWrite(path string, err error) error {
	return gerror.Wrap(err, ErrCodeFileWrite, "failed to write file").WithDetails("path", path)
}

func ErrFileRead(path string, err error) error {
	return gerror.Wrap(err, ErrCodeFileRead, "failed to read file").WithDetails("path", path)
}

func ErrYAMLParse(path string, err error) error {
	return gerror.Wrap(err, ErrCodeYAMLParse, "failed to parse YAML").WithDetails("path", path)
}
func ErrValidation(message string) error { return gerror.New(ErrCodeValidation, message, nil) }
func ErrTimeout(operation string, duration string) error {
	return gerror.New(ErrCodeTimeout, "operation timed out", nil).WithDetails("operation", operation).WithDetails("timeout", duration)
}

func ErrPermission(path string, err error) error {
	return gerror.Wrap(err, ErrCodePermission, "permission denied").WithDetails("path", path)
}

func ErrInvalidPath(path string, reason string) error {
	return gerror.New(ErrCodeInvalidPath, "invalid file path", nil).WithDetails("path", path).WithDetails("reason", reason)
}

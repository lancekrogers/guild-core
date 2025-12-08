package scaffoldlite

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

type semanticValidator struct {
	templateChecker TemplateChecker
	pathValidator   PathValidator
	variableChecker VariableChecker
}

func newSemanticValidator() *semanticValidator {
	return &semanticValidator{
		templateChecker: &templateChecker{},
		pathValidator:   &pathValidator{},
		variableChecker: &variableChecker{},
	}
}

func (sv *semanticValidator) Validate(ctx context.Context, recipe *Recipe) []ValidationError {
	var errors []ValidationError
	if sv.templateChecker != nil {
		templateErrors := sv.templateChecker.CheckTemplates(ctx, recipe)
		errors = append(errors, templateErrors...)
	}
	pathErrors := sv.pathValidator.CheckPaths(recipe)
	errors = append(errors, pathErrors...)
	varErrors := sv.variableChecker.CheckVariables(recipe)
	errors = append(errors, varErrors...)
	versionErrors := sv.validateScaffoldVersion(recipe)
	errors = append(errors, versionErrors...)
	dirErrors := sv.validateTemplatesDirectory(recipe)
	errors = append(errors, dirErrors...)
	return errors
}

func (sv *semanticValidator) SetTemplateFS(fsys fs.FS) {
	if tc, ok := sv.templateChecker.(*templateChecker); ok {
		tc.templateFS = fsys
	}
}

func (sv *semanticValidator) validateScaffoldVersion(recipe *Recipe) []ValidationError {
	var errors []ValidationError
	versionRegex := regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	if recipe.ScaffoldVersion == "" || !versionRegex.MatchString(recipe.ScaffoldVersion) {
		errors = append(errors, ValidationError{Field: "scaffold_version", Message: "must be a valid semantic version (e.g., '1.0.0')", Value: recipe.ScaffoldVersion, Code: ErrCodeValidation})
	}
	return errors
}

func (sv *semanticValidator) validateTemplatesDirectory(recipe *Recipe) []ValidationError {
	var errors []ValidationError
	if strings.ContainsAny(recipe.TemplatesDir, `<>:"|?*`) {
		errors = append(errors, ValidationError{Field: "templates_dir", Message: "contains invalid characters", Value: recipe.TemplatesDir, Code: ErrCodeInvalidPath})
	}
	if strings.Contains(recipe.TemplatesDir, "..") {
		errors = append(errors, ValidationError{Field: "templates_dir", Message: "path traversal not allowed", Value: recipe.TemplatesDir, Code: ErrCodeInvalidPath})
	}
	return errors
}

type TemplateChecker interface {
	CheckTemplates(ctx context.Context, recipe *Recipe) []ValidationError
}

type templateChecker struct{ templateFS fs.FS }

func (tc *templateChecker) CheckTemplates(ctx context.Context, recipe *Recipe) []ValidationError {
	var errors []ValidationError
	if tc.templateFS == nil {
		return errors
	}
	templatesSeen := make(map[string]bool)
	for i, file := range recipe.Files {
		select {
		case <-ctx.Done():
			errors = append(errors, ValidationError{Field: "validation", Message: "validation cancelled", Code: "VALIDATION_CANCELLED"})
			return errors
		default:
		}
		templatePath := filepath.Join(recipe.TemplatesDir, file.Template)
		if _, err := fs.Stat(tc.templateFS, templatePath); err != nil {
			errors = append(errors, ValidationError{Field: fmt.Sprintf("files[%d].template", i), Message: fmt.Sprintf("template not found: %s", file.Template), Value: file.Template, Code: ErrCodeTemplateNotFound})
			continue
		}
		if !templatesSeen[file.Template] {
			templatesSeen[file.Template] = true
			if syntaxErrors := tc.validateTemplateSyntax(templatePath, file.Template); len(syntaxErrors) > 0 {
				errors = append(errors, syntaxErrors...)
			}
		}
	}
	return errors
}

func (tc *templateChecker) validateTemplateSyntax(templatePath, templateName string) []ValidationError {
	var errors []ValidationError
	content, err := fs.ReadFile(tc.templateFS, templatePath)
	if err != nil {
		errors = append(errors, ValidationError{Field: "template", Message: fmt.Sprintf("failed to read template: %v", err), Value: templateName, Code: ErrCodeFileRead})
		return errors
	}
	tmpl := template.New(templateName).Funcs(getValidationTemplateFuncMap())
	if _, err := tmpl.Parse(string(content)); err != nil {
		errors = append(errors, ValidationError{Field: "template", Message: fmt.Sprintf("invalid template syntax: %v", err), Value: templateName, Code: ErrCodeTemplateRender})
	}
	return errors
}

func getValidationTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"default": func(def, val any) any {
			if val == nil {
				return def
			}
			return val
		},
	}
}

// Path and variable validators (minimal implementations)
type PathValidator interface {
	CheckPaths(recipe *Recipe) []ValidationError
}
type VariableChecker interface {
	CheckVariables(recipe *Recipe) []ValidationError
}

type pathValidator struct{}

func (p *pathValidator) CheckPaths(recipe *Recipe) []ValidationError { return nil }

type variableChecker struct{}

func (v *variableChecker) CheckVariables(recipe *Recipe) []ValidationError { return nil }

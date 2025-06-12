package code

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/tools"
)

// DependenciesTool analyzes project dependencies
type DependenciesTool struct {
	*tools.BaseTool
}

// DependenciesParams represents the input parameters for dependency analysis
type DependenciesParams struct {
	ProjectPath  string `json:"project_path,omitempty"`  // Path to project root (auto-detect if empty)
	ShowTree     bool   `json:"show_tree,omitempty"`     // Show dependency tree
	CheckUpdates bool   `json:"check_updates,omitempty"` // Check for outdated packages
	OnlyDirect   bool   `json:"only_direct,omitempty"`   // Only show direct dependencies
	Format       string `json:"format,omitempty"`        // Output format: text, json, summary
}

// DependenciesResult represents the result of dependency analysis
type DependenciesResult struct {
	ProjectPath     string             `json:"project_path"`
	ProjectType     string             `json:"project_type"` // go, python, node, etc.
	TotalDirect     int                `json:"total_direct"`
	TotalTransitive int                `json:"total_transitive"`
	Dependencies    []*Dependency      `json:"dependencies"`
	OutdatedCount   int                `json:"outdated_count"`
	SecurityIssues  int                `json:"security_issues"`
	Summary         *DependencySummary `json:"summary"`
	Errors          []string           `json:"errors,omitempty"`
}

// DependencySummary provides a summary of dependency analysis
type DependencySummary struct {
	TotalSize       int64          `json:"total_size"`
	AverageAge      int            `json:"average_age_days"`
	LicenseTypes    map[string]int `json:"license_types"`
	TopCategories   map[string]int `json:"top_categories"`
	RiskScore       float64        `json:"risk_score"` // 0-10, higher is riskier
	Recommendations []string       `json:"recommendations"`
}

// NewDependenciesTool creates a new dependencies analysis tool
func NewDependenciesTool() *DependenciesTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"project_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to project root (auto-detected if not specified)",
			},
			"show_tree": map[string]interface{}{
				"type":        "boolean",
				"description": "Show full dependency tree with sub-dependencies",
			},
			"check_updates": map[string]interface{}{
				"type":        "boolean",
				"description": "Check for outdated packages and available updates",
			},
			"only_direct": map[string]interface{}{
				"type":        "boolean",
				"description": "Only analyze direct dependencies, skip transitive ones",
			},
			"format": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"text", "json", "summary"},
				"description": "Output format for results",
			},
		},
	}

	examples := []string{
		`{"project_path": ".", "format": "summary"}`,
		`{"project_path": "/path/to/project", "show_tree": true}`,
		`{"check_updates": true, "only_direct": true}`,
		`{"project_path": ".", "show_tree": true, "format": "json"}`,
		`{"project_path": ".", "check_updates": true, "format": "summary"}`,
	}

	baseTool := tools.NewBaseTool(
		"dependencies",
		"Analyze project dependencies, show dependency trees, check for updates, and identify security vulnerabilities.",
		schema,
		"code",
		false,
		examples,
	)

	return &DependenciesTool{
		BaseTool: baseTool,
	}
}

// Execute runs the dependencies tool with the given input
func (t *DependenciesTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params DependenciesParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input").
			WithComponent("dependencies_tool").
			WithOperation("execute")
	}

	// Default project path to current directory
	if params.ProjectPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get current directory").
				WithComponent("dependencies_tool").
				WithOperation("execute")
		}
		params.ProjectPath = cwd
	}

	// Ensure project path exists
	if _, err := os.Stat(params.ProjectPath); os.IsNotExist(err) {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "project path does not exist: %s", params.ProjectPath).
			WithComponent("dependencies_tool").
			WithOperation("execute")
	}

	// Detect project type
	projectType := t.detectProjectType(params.ProjectPath)
	if projectType == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "could not detect project type (no go.mod, package.json, requirements.txt, etc.)", nil).
			WithComponent("dependencies_tool").
			WithOperation("execute")
	}

	// Analyze dependencies based on project type
	result, err := t.analyzeDependencies(ctx, params, projectType)
	if err != nil {
		return tools.NewToolResult("", map[string]string{
			"project_path": params.ProjectPath,
			"project_type": projectType,
		}, err, nil), err
	}

	// Format output
	output, err := t.formatResult(result, params.Format)
	if err != nil {
		return tools.NewToolResult("", map[string]string{
			"project_path": params.ProjectPath,
			"project_type": projectType,
		}, err, nil), err
	}

	metadata := map[string]string{
		"project_path":     params.ProjectPath,
		"project_type":     projectType,
		"total_direct":     fmt.Sprintf("%d", result.TotalDirect),
		"total_transitive": fmt.Sprintf("%d", result.TotalTransitive),
		"outdated_count":   fmt.Sprintf("%d", result.OutdatedCount),
	}

	extraData := map[string]interface{}{
		"result": result,
	}

	return tools.NewToolResult(output, metadata, nil, extraData), nil
}

// detectProjectType detects the project type based on files in the project directory
func (t *DependenciesTool) detectProjectType(projectPath string) string {
	// Check for Go project
	if _, err := os.Stat(filepath.Join(projectPath, "go.mod")); err == nil {
		return "go"
	}

	// Check for Node.js project
	if _, err := os.Stat(filepath.Join(projectPath, "package.json")); err == nil {
		return "node"
	}

	// Check for Python project
	for _, file := range []string{"requirements.txt", "pyproject.toml", "setup.py", "Pipfile"} {
		if _, err := os.Stat(filepath.Join(projectPath, file)); err == nil {
			return "python"
		}
	}

	// Check for Rust project
	if _, err := os.Stat(filepath.Join(projectPath, "Cargo.toml")); err == nil {
		return "rust"
	}

	// Check for Java project
	for _, file := range []string{"pom.xml", "build.gradle", "build.gradle.kts"} {
		if _, err := os.Stat(filepath.Join(projectPath, file)); err == nil {
			return "java"
		}
	}

	return ""
}

// analyzeDependencies analyzes dependencies based on project type
func (t *DependenciesTool) analyzeDependencies(ctx context.Context, params DependenciesParams, projectType string) (*DependenciesResult, error) {
	switch projectType {
	case "go":
		return t.analyzeGoDependencies(ctx, params)
	case "node":
		return t.analyzeNodeDependencies(ctx, params)
	case "python":
		return t.analyzePythonDependencies(ctx, params)
	case "rust":
		return t.analyzeRustDependencies(ctx, params)
	default:
		return nil, gerror.Newf(gerror.ErrCodeNotImplemented, "dependency analysis not implemented for project type: %s", projectType).
			WithComponent("dependencies_tool").
			WithOperation("analyze_dependencies")
	}
}

// analyzeGoDependencies analyzes Go module dependencies
func (t *DependenciesTool) analyzeGoDependencies(ctx context.Context, params DependenciesParams) (*DependenciesResult, error) {
	result := &DependenciesResult{
		ProjectPath: params.ProjectPath,
		ProjectType: "go",
		Summary:     &DependencySummary{LicenseTypes: make(map[string]int), TopCategories: make(map[string]int)},
	}

	// Change to project directory
	oldDir, err := os.Getwd()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get current directory").
			WithComponent("dependencies_tool").
			WithOperation("analyze_go_dependencies")
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(params.ProjectPath); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to change to project directory").
			WithComponent("dependencies_tool").
			WithOperation("analyze_go_dependencies")
	}

	// Get direct dependencies from go.mod
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-json", "all")
	output, err := cmd.Output()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to run 'go list -m -json all'").
			WithComponent("dependencies_tool").
			WithOperation("analyze_go_dependencies")
	}

	// Parse go list output
	dependencies, err := t.parseGoListOutput(string(output), params.OnlyDirect)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse go list output").
			WithComponent("dependencies_tool").
			WithOperation("analyze_go_dependencies")
	}

	result.Dependencies = dependencies

	// Count direct vs transitive
	for _, dep := range dependencies {
		if dep.Type == "direct" {
			result.TotalDirect++
		} else {
			result.TotalTransitive++
		}
	}

	// Check for updates if requested
	if params.CheckUpdates {
		err := t.checkGoUpdates(ctx, result)
		if err != nil {
			result.Errors = append(result.Errors, "Failed to check for updates: "+err.Error())
		}
	}

	// Generate summary
	t.generateSummary(result)

	return result, nil
}

// analyzeNodeDependencies analyzes Node.js package dependencies
func (t *DependenciesTool) analyzeNodeDependencies(ctx context.Context, params DependenciesParams) (*DependenciesResult, error) {
	result := &DependenciesResult{
		ProjectPath: params.ProjectPath,
		ProjectType: "node",
		Summary:     &DependencySummary{LicenseTypes: make(map[string]int), TopCategories: make(map[string]int)},
	}

	// Read package.json
	packageJsonPath := filepath.Join(params.ProjectPath, "package.json")
	content, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to read package.json").
			WithComponent("dependencies_tool").
			WithOperation("analyze_node_dependencies")
	}

	// Parse package.json
	var packageJson map[string]interface{}
	if err := json.Unmarshal(content, &packageJson); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse package.json").
			WithComponent("dependencies_tool").
			WithOperation("analyze_node_dependencies")
	}

	// Extract dependencies
	dependencies := t.extractNodeDependencies(packageJson, params.OnlyDirect)
	result.Dependencies = dependencies

	// Count dependencies
	for _, dep := range dependencies {
		if dep.Type == "direct" {
			result.TotalDirect++
		} else {
			result.TotalTransitive++
		}
	}

	// Check for updates if requested
	if params.CheckUpdates {
		err := t.checkNodeUpdates(ctx, params.ProjectPath, result)
		if err != nil {
			result.Errors = append(result.Errors, "Failed to check for updates: "+err.Error())
		}
	}

	// Generate summary
	t.generateSummary(result)

	return result, nil
}

// analyzePythonDependencies analyzes Python package dependencies
func (t *DependenciesTool) analyzePythonDependencies(ctx context.Context, params DependenciesParams) (*DependenciesResult, error) {
	result := &DependenciesResult{
		ProjectPath: params.ProjectPath,
		ProjectType: "python",
		Summary:     &DependencySummary{LicenseTypes: make(map[string]int), TopCategories: make(map[string]int)},
	}

	// Try different Python dependency files
	var dependencies []*Dependency
	var err error

	// Check for requirements.txt
	reqPath := filepath.Join(params.ProjectPath, "requirements.txt")
	if _, statErr := os.Stat(reqPath); statErr == nil {
		dependencies, err = t.parseRequirementsTxt(reqPath)
		if err != nil {
			result.Errors = append(result.Errors, "Failed to parse requirements.txt: "+err.Error())
		}
	}

	// Check for pyproject.toml
	if len(dependencies) == 0 {
		pyprojectPath := filepath.Join(params.ProjectPath, "pyproject.toml")
		if _, statErr := os.Stat(pyprojectPath); statErr == nil {
			// This would require a TOML parser - simplified for now
			result.Errors = append(result.Errors, "pyproject.toml parsing not yet implemented")
		}
	}

	if len(dependencies) == 0 {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no dependency files found (requirements.txt, pyproject.toml, etc.)", nil).
			WithComponent("dependencies_tool").
			WithOperation("analyze_python_dependencies")
	}

	result.Dependencies = dependencies
	result.TotalDirect = len(dependencies)

	// Generate summary
	t.generateSummary(result)

	return result, nil
}

// analyzeRustDependencies analyzes Rust Cargo dependencies
func (t *DependenciesTool) analyzeRustDependencies(ctx context.Context, params DependenciesParams) (*DependenciesResult, error) {
	// Placeholder implementation for Rust
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "Rust dependency analysis not yet implemented", nil).
		WithComponent("dependencies_tool").
		WithOperation("analyze_rust_dependencies")
}

// parseGoListOutput parses the output of 'go list -m -json all'
func (t *DependenciesTool) parseGoListOutput(output string, onlyDirect bool) ([]*Dependency, error) {
	var dependencies []*Dependency

	// Split JSON objects (each line is a separate JSON object)
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var module map[string]interface{}
		if err := json.Unmarshal([]byte(line), &module); err != nil {
			continue // Skip invalid JSON
		}

		path, ok := module["Path"].(string)
		if !ok || path == "" {
			continue
		}

		// Skip the main module (first entry)
		if i == 0 {
			continue
		}

		version, _ := module["Version"].(string)
		indirect, _ := module["Indirect"].(bool)

		depType := "direct"
		if indirect {
			depType = "transitive"
		}

		// Skip transitive dependencies if onlyDirect is true
		if onlyDirect && depType == "transitive" {
			continue
		}

		dep := &Dependency{
			Name:    path,
			Version: version,
			Type:    depType,
			Source:  "go modules",
		}

		dependencies = append(dependencies, dep)
	}

	return dependencies, nil
}

// extractNodeDependencies extracts dependencies from package.json
func (t *DependenciesTool) extractNodeDependencies(packageJson map[string]interface{}, onlyDirect bool) []*Dependency {
	var dependencies []*Dependency

	// Extract direct dependencies
	if deps, ok := packageJson["dependencies"].(map[string]interface{}); ok {
		for name, version := range deps {
			if versionStr, ok := version.(string); ok {
				dep := &Dependency{
					Name:    name,
					Version: versionStr,
					Type:    "direct",
					Source:  "npm",
				}
				dependencies = append(dependencies, dep)
			}
		}
	}

	// Extract dev dependencies
	if devDeps, ok := packageJson["devDependencies"].(map[string]interface{}); ok {
		for name, version := range devDeps {
			if versionStr, ok := version.(string); ok {
				dep := &Dependency{
					Name:    name,
					Version: versionStr,
					Type:    "dev",
					Source:  "npm",
				}
				dependencies = append(dependencies, dep)
			}
		}
	}

	return dependencies
}

// parseRequirementsTxt parses a Python requirements.txt file
func (t *DependenciesTool) parseRequirementsTxt(filePath string) ([]*Dependency, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var dependencies []*Dependency
	lines := strings.Split(string(content), "\n")

	// Simple regex for requirement parsing (simplified)
	reqRegex := regexp.MustCompile(`^([a-zA-Z0-9\-_]+)([><=!]+)?([0-9\.]+.*)?`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := reqRegex.FindStringSubmatch(line)
		if len(matches) >= 2 {
			name := matches[1]
			version := ""
			if len(matches) >= 4 && matches[3] != "" {
				version = matches[2] + matches[3]
			}

			dep := &Dependency{
				Name:    name,
				Version: version,
				Type:    "direct",
				Source:  "pip",
			}
			dependencies = append(dependencies, dep)
		}
	}

	return dependencies, nil
}

// checkGoUpdates checks for available updates to Go modules
func (t *DependenciesTool) checkGoUpdates(ctx context.Context, result *DependenciesResult) error {
	// Run 'go list -u -m all' to check for updates
	cmd := exec.CommandContext(ctx, "go", "list", "-u", "-m", "-json", "all")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	// Parse the output to find outdated packages
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var module map[string]interface{}
		if err := json.Unmarshal([]byte(line), &module); err != nil {
			continue
		}

		path, _ := module["Path"].(string)
		update, hasUpdate := module["Update"]

		if hasUpdate && update != nil {
			if updateMap, ok := update.(map[string]interface{}); ok {
				latestVersion, _ := updateMap["Version"].(string)

				// Find the corresponding dependency and update it
				for _, dep := range result.Dependencies {
					if dep.Name == path {
						dep.LatestVersion = latestVersion
						dep.IsOutdated = true
						result.OutdatedCount++
						break
					}
				}
			}
		}
	}

	return nil
}

// checkNodeUpdates checks for available updates to Node.js packages
func (t *DependenciesTool) checkNodeUpdates(ctx context.Context, projectPath string, result *DependenciesResult) error {
	// This would typically use 'npm outdated' or similar
	// Simplified implementation for now
	result.Errors = append(result.Errors, "Node.js update checking not yet implemented")
	return nil
}

// generateSummary generates summary statistics for dependencies
func (t *DependenciesTool) generateSummary(result *DependenciesResult) {
	summary := result.Summary

	// Calculate risk score based on various factors
	riskScore := 0.0

	// Factor 1: Number of dependencies (more = higher risk)
	depCount := float64(len(result.Dependencies))
	riskScore += (depCount / 100.0) * 2.0 // 2 points per 100 deps

	// Factor 2: Outdated packages
	if result.OutdatedCount > 0 {
		riskScore += float64(result.OutdatedCount) * 0.5
	}

	// Factor 3: Security issues
	riskScore += float64(result.SecurityIssues) * 2.0

	// Cap at 10
	if riskScore > 10.0 {
		riskScore = 10.0
	}

	summary.RiskScore = riskScore

	// Generate recommendations
	if result.OutdatedCount > 5 {
		summary.Recommendations = append(summary.Recommendations, "Consider updating outdated packages")
	}
	if depCount > 50 {
		summary.Recommendations = append(summary.Recommendations, "Large number of dependencies - consider dependency cleanup")
	}
	if riskScore > 7.0 {
		summary.Recommendations = append(summary.Recommendations, "High risk score - review dependencies for security and maintenance")
	}
	if len(summary.Recommendations) == 0 {
		summary.Recommendations = append(summary.Recommendations, "Dependencies look healthy")
	}
}

// formatResult formats the dependency analysis result
func (t *DependenciesTool) formatResult(result *DependenciesResult, format string) (string, error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil

	case "summary":
		return t.formatSummary(result), nil

	default: // "text"
		return t.formatText(result), nil
	}
}

// formatSummary formats a summary view of dependencies
func (t *DependenciesTool) formatSummary(result *DependenciesResult) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("Dependency Analysis Summary\n"))
	output.WriteString(fmt.Sprintf("Project: %s (%s)\n", result.ProjectPath, result.ProjectType))
	output.WriteString(fmt.Sprintf("Direct Dependencies: %d\n", result.TotalDirect))
	output.WriteString(fmt.Sprintf("Transitive Dependencies: %d\n", result.TotalTransitive))

	if result.OutdatedCount > 0 {
		output.WriteString(fmt.Sprintf("Outdated Packages: %d\n", result.OutdatedCount))
	}

	if result.SecurityIssues > 0 {
		output.WriteString(fmt.Sprintf("Security Issues: %d\n", result.SecurityIssues))
	}

	output.WriteString(fmt.Sprintf("Risk Score: %.1f/10\n", result.Summary.RiskScore))

	if len(result.Summary.Recommendations) > 0 {
		output.WriteString("\nRecommendations:\n")
		for _, rec := range result.Summary.Recommendations {
			output.WriteString(fmt.Sprintf("- %s\n", rec))
		}
	}

	if len(result.Errors) > 0 {
		output.WriteString("\nWarnings:\n")
		for _, err := range result.Errors {
			output.WriteString(fmt.Sprintf("- %s\n", err))
		}
	}

	return output.String()
}

// formatText formats a detailed text view of dependencies
func (t *DependenciesTool) formatText(result *DependenciesResult) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("Dependencies for %s (%s)\n", result.ProjectPath, result.ProjectType))
	output.WriteString(fmt.Sprintf("Total: %d direct, %d transitive\n\n", result.TotalDirect, result.TotalTransitive))

	// Group by type
	directDeps := []*Dependency{}
	transitiveDeps := []*Dependency{}
	devDeps := []*Dependency{}

	for _, dep := range result.Dependencies {
		switch dep.Type {
		case "direct":
			directDeps = append(directDeps, dep)
		case "transitive":
			transitiveDeps = append(transitiveDeps, dep)
		case "dev":
			devDeps = append(devDeps, dep)
		}
	}

	// Output direct dependencies
	if len(directDeps) > 0 {
		output.WriteString(fmt.Sprintf("Direct Dependencies (%d):\n", len(directDeps)))
		for _, dep := range directDeps {
			status := ""
			if dep.IsOutdated {
				status = fmt.Sprintf(" (outdated, latest: %s)", dep.LatestVersion)
			}
			output.WriteString(fmt.Sprintf("  %s %s%s\n", dep.Name, dep.Version, status))
		}
		output.WriteString("\n")
	}

	// Output dev dependencies
	if len(devDeps) > 0 {
		output.WriteString(fmt.Sprintf("Development Dependencies (%d):\n", len(devDeps)))
		for _, dep := range devDeps {
			output.WriteString(fmt.Sprintf("  %s %s\n", dep.Name, dep.Version))
		}
		output.WriteString("\n")
	}

	// Output transitive dependencies (limited)
	if len(transitiveDeps) > 0 {
		output.WriteString(fmt.Sprintf("Transitive Dependencies (%d):\n", len(transitiveDeps)))
		for i, dep := range transitiveDeps {
			if i >= 10 { // Limit output
				output.WriteString(fmt.Sprintf("  ... and %d more\n", len(transitiveDeps)-10))
				break
			}
			output.WriteString(fmt.Sprintf("  %s %s\n", dep.Name, dep.Version))
		}
		output.WriteString("\n")
	}

	// Add summary
	output.WriteString(t.formatSummary(result))

	return output.String()
}

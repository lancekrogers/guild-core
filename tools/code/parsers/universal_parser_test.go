package parsers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/guild-ventures/guild-core/tools/code"
	"github.com/guild-ventures/guild-core/tools/code/parsers"
)

func TestUniversalParser_JavaScript(t *testing.T) {
	parser, err := parsers.NewUniversalParser(code.LanguageJavaScript)
	require.NoError(t, err)

	content := []byte(`
// Main function
function calculateTotal(items) {
    return items.reduce((sum, item) => sum + item.price, 0);
}

class ShoppingCart {
    constructor() {
        this.items = [];
    }

    addItem(item) {
        this.items.push(item);
    }

    getTotal() {
        return calculateTotal(this.items);
    }
}

export { ShoppingCart, calculateTotal };
`)

	result, err := parser.Parse(context.Background(), "test.js", content)
	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, code.LanguageJavaScript, result.Language)

	// Test basic parsing
	functions, err := parser.GetFunctions(result)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(functions), 1)

	classes, err := parser.GetClasses(result)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(classes), 1)
}

func TestUniversalParser_Rust(t *testing.T) {
	parser, err := parsers.NewUniversalParser(code.LanguageRust)
	require.NoError(t, err)

	content := []byte(`
use std::collections::HashMap;

fn main() {
    let mut scores = HashMap::new();
    scores.insert("Blue", 10);
    scores.insert("Yellow", 50);
}

struct Point {
    x: f64,
    y: f64,
}

impl Point {
    fn new(x: f64, y: f64) -> Self {
        Point { x, y }
    }

    fn distance(&self, other: &Point) -> f64 {
        ((self.x - other.x).powi(2) + (self.y - other.y).powi(2)).sqrt()
    }
}
`)

	result, err := parser.Parse(context.Background(), "test.rs", content)
	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, code.LanguageRust, result.Language)
}

func TestUniversalParser_Ruby(t *testing.T) {
	parser, err := parsers.NewUniversalParser(code.LanguageRuby)
	require.NoError(t, err)

	content := []byte(`
class Person
  attr_accessor :name, :age

  def initialize(name, age)
    @name = name
    @age = age
  end

  def greeting
    "Hello, my name is #{@name}"
  end
end

def calculate_tax(income)
  case income
  when 0..10_000
    income * 0.1
  when 10_001..50_000
    income * 0.2
  else
    income * 0.3
  end
end
`)

	result, err := parser.Parse(context.Background(), "test.rb", content)
	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, code.LanguageRuby, result.Language)
}

func TestUniversalParser_Multiple_Languages(t *testing.T) {
	tests := []struct {
		name     string
		language code.Language
		filename string
		content  string
	}{
		{
			name:     "Bash Script",
			language: code.LanguageBash,
			filename: "test.sh",
			content: `#!/bin/bash
function setup() {
    echo "Setting up environment"
}

setup
`,
		},
		{
			name:     "CSS Stylesheet",
			language: code.LanguageCSS,
			filename: "test.css",
			content: `.container {
    display: flex;
    justify-content: center;
}`,
		},
		{
			name:     "HTML Document",
			language: code.LanguageHTML,
			filename: "test.html",
			content: `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><h1>Hello World</h1></body>
</html>`,
		},
		{
			name:     "SQL Query",
			language: code.LanguageSQL,
			filename: "test.sql",
			content: `SELECT name, age FROM users WHERE age > 18 ORDER BY name;`,
		},
		{
			name:     "YAML Config",
			language: code.LanguageYAML,
			filename: "test.yaml",
			content: `name: test-app
version: 1.0.0
services:
  - web
  - api`,
		},
		{
			name:     "Dockerfile",
			language: code.LanguageDockerfile,
			filename: "Dockerfile",
			content: `FROM golang:1.21
WORKDIR /app
COPY . .
RUN go build -o main .
CMD ["./main"]
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := parsers.NewUniversalParser(tt.language)
			require.NoError(t, err)

			result, err := parser.Parse(context.Background(), tt.filename, []byte(tt.content))
			require.NoError(t, err)
			assert.Empty(t, result.Errors)
			assert.Equal(t, tt.language, result.Language)
			assert.NotNil(t, result.AST)
		})
	}
}

func TestCreateParser_SpecialCases(t *testing.T) {
	// Test that Go and Python use their custom parsers
	goParser, err := parsers.CreateParser(code.LanguageGo)
	require.NoError(t, err)
	_, ok := goParser.(*parsers.GoParser)
	assert.True(t, ok, "Go should use custom GoParser")

	pyParser, err := parsers.CreateParser(code.LanguagePython)
	require.NoError(t, err)
	_, ok = pyParser.(*parsers.PythonTreeSitterParser)
	assert.True(t, ok, "Python should use custom PythonTreeSitterParser")

	// Test that other languages use UniversalParser
	jsParser, err := parsers.CreateParser(code.LanguageJavaScript)
	require.NoError(t, err)
	_, ok = jsParser.(*parsers.UniversalParser)
	assert.True(t, ok, "JavaScript should use UniversalParser")
}

func TestGetSupportedLanguages(t *testing.T) {
	languages := parsers.GetSupportedLanguages()
	assert.GreaterOrEqual(t, len(languages), 30, "Should support at least 30 languages")
	
	// Check some key languages are present
	hasGo, hasPython, hasJS, hasRust := false, false, false, false
	for _, lang := range languages {
		switch lang {
		case code.LanguageGo:
			hasGo = true
		case code.LanguagePython:
			hasPython = true
		case code.LanguageJavaScript:
			hasJS = true
		case code.LanguageRust:
			hasRust = true
		}
	}
	
	assert.True(t, hasGo, "Should support Go")
	assert.True(t, hasPython, "Should support Python")
	assert.True(t, hasJS, "Should support JavaScript")
	assert.True(t, hasRust, "Should support Rust")
}

func TestGetLanguageExtensions(t *testing.T) {
	// Test Go extensions
	goExts := parsers.GetLanguageExtensions(code.LanguageGo)
	assert.Contains(t, goExts, ".go")

	// Test Python extensions
	pyExts := parsers.GetLanguageExtensions(code.LanguagePython)
	assert.Contains(t, pyExts, ".py")
	assert.Contains(t, pyExts, ".pyw")

	// Test TypeScript extensions
	tsExts := parsers.GetLanguageExtensions(code.LanguageTypeScript)
	assert.Contains(t, tsExts, ".ts")
	assert.Contains(t, tsExts, ".tsx")
}
// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package parsers_example demonstrates how to use the universal tree-sitter parser
//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lancekrogers/guild/tools/code"
	"github.com/lancekrogers/guild/tools/code/parsers"
)

func main() {
	// Example: Parse JavaScript code
	jsCode := `
function greet(name) {
    return "Hello, " + name + "!";
}

class Person {
    constructor(name) {
        this.name = name;
    }
    
    sayHello() {
        return greet(this.name);
    }
}
`

	// Create a JavaScript parser
	jsParser, err := parsers.CreateParser(code.LanguageJavaScript)
	if err != nil {
		log.Fatal(err)
	}

	// Parse the code
	result, err := jsParser.Parse(context.Background(), "example.js", []byte(jsCode))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Parsed %s successfully!\n", result.Language)
	fmt.Printf("Parse errors: %d\n", len(result.Errors))

	// Example: Parse Python code
	pyCode := `
def calculate_factorial(n: int) -> int:
    """Calculate factorial recursively"""
    if n <= 1:
        return 1
    return n * calculate_factorial(n - 1)

class Calculator:
    @staticmethod
    def factorial(n):
        return calculate_factorial(n)
`

	// Create a Python parser (uses custom implementation)
	pyParser, err := parsers.CreateParser(code.LanguagePython)
	if err != nil {
		log.Fatal(err)
	}

	// Parse and extract functions
	pyResult, err := pyParser.Parse(context.Background(), "example.py", []byte(pyCode))
	if err != nil {
		log.Fatal(err)
	}

	functions, err := pyParser.GetFunctions(pyResult)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nPython functions found: %d\n", len(functions))
	for _, fn := range functions {
		fmt.Printf("- %s (lines %d-%d)\n", fn.Name, fn.StartLine, fn.EndLine)
	}

	// Show all supported languages
	fmt.Println("\nAll supported languages:")
	languages := parsers.GetSupportedLanguages()
	for i, lang := range languages {
		fmt.Printf("%2d. %s\n", i+1, lang)
	}
}

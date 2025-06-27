// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parsers_test

import (
	"context"
	"testing"

	"github.com/lancekrogers/guild/tools/code/parsers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPythonParser_Functions(t *testing.T) {
	parser := parsers.NewPythonParser()

	content := []byte(`
def simple_function():
    """A simple function"""
    return 42

async def async_function(param1: str, param2: int = 10) -> dict:
    """An async function with type hints"""
    return {"result": param1}

@decorator
def decorated_function(*args, **kwargs):
    pass
`)

	result, err := parser.Parse(context.Background(), "test.py", content)
	require.NoError(t, err)
	assert.Empty(t, result.Errors)

	functions, err := parser.GetFunctions(result)
	require.NoError(t, err)
	assert.Len(t, functions, 3)

	// Test simple function
	assert.Equal(t, "simple_function", functions[0].Name)
	assert.Len(t, functions[0].Parameters, 0)
	assert.Contains(t, functions[0].DocString, "simple function")

	// Test async function
	assert.Equal(t, "async_function", functions[1].Name)
	assert.True(t, functions[1].IsStatic) // Using IsStatic for async
	assert.Len(t, functions[1].Parameters, 2)
	assert.Equal(t, "param1", functions[1].Parameters[0].Name)
	assert.Equal(t, "str", functions[1].Parameters[0].Type)
	assert.Equal(t, "param2", functions[1].Parameters[1].Name)
	assert.Equal(t, "int", functions[1].Parameters[1].Type)
	assert.Equal(t, "dict", functions[1].ReturnType)

	// Test decorated function
	assert.Equal(t, "decorated_function", functions[2].Name)
	assert.Len(t, functions[2].Decorators, 1)
}

func TestPythonParser_Classes(t *testing.T) {
	parser := parsers.NewPythonParser()

	content := []byte(`
class SimpleClass:
    """A simple class"""
    def __init__(self):
        self.value = 0
    
    def method(self, x):
        return x * 2

class DerivedClass(SimpleClass):
    def method(self, x):
        return super().method(x) + 1
`)

	result, err := parser.Parse(context.Background(), "test.py", content)
	require.NoError(t, err)

	classes, err := parser.GetClasses(result)
	require.NoError(t, err)
	assert.Len(t, classes, 2)

	// Test simple class
	assert.Equal(t, "SimpleClass", classes[0].Name)
	assert.Contains(t, classes[0].DocString, "simple class")
	assert.Len(t, classes[0].Methods, 2)
	assert.Equal(t, "__init__", classes[0].Methods[0].Name)
	assert.Equal(t, "method", classes[0].Methods[1].Name)

	// Test derived class
	assert.Equal(t, "DerivedClass", classes[1].Name)
	assert.Equal(t, []string{"SimpleClass"}, classes[1].BaseClasses)
	assert.Len(t, classes[1].Methods, 1)
}

func TestPythonParser_Imports(t *testing.T) {
	parser := parsers.NewPythonParser()

	content := []byte(`
import os
import sys as system
from datetime import datetime
from typing import List, Dict
from collections.abc import Mapping as ABCMapping
`)

	result, err := parser.Parse(context.Background(), "test.py", content)
	require.NoError(t, err)

	imports, err := parser.GetImports(result)
	require.NoError(t, err)
	assert.Len(t, imports, 6)

	// Check specific imports
	importPaths := make(map[string]string)
	for _, imp := range imports {
		importPaths[imp.Path] = imp.Alias
	}

	assert.Equal(t, "", importPaths["os"])
	assert.Equal(t, "system", importPaths["sys"])
	assert.Equal(t, "", importPaths["datetime.datetime"])
	assert.Equal(t, "", importPaths["typing.List"])
	assert.Equal(t, "ABCMapping", importPaths["collections.abc.Mapping"])
}

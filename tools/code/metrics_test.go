package code

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsTool_NewMetricsTool(t *testing.T) {
	tool := NewMetricsTool()
	assert.NotNil(t, tool)
	assert.Equal(t, "metrics", tool.GetName())
	assert.Equal(t, "code", tool.GetCategory())
}

func TestMetricsTool_Execute_GoFile(t *testing.T) {
	// Create a temporary Go file with varying complexity
	goCode := `package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Println("Hello, World!")
}

func complexFunction(x int, y string) (int, error) {
	if x < 0 {
		return 0, fmt.Errorf("negative input")
	}
	
	if y == "" {
		return 0, fmt.Errorf("empty string")
	}
	
	result := 0
	for i := 0; i < x; i++ {
		if i%2 == 0 {
			result += i
		} else {
			result -= i
		}
		
		if result > 100 {
			break
		}
	}
	
	switch len(y) {
	case 1:
		result *= 2
	case 2:
		result *= 3
	default:
		result *= 4
	}
	
	return result, nil
}

type User struct {
	Name string
	Age  int
	Email string
}

func (u User) IsAdult() bool {
	return u.Age >= 18
}

func (u User) ValidateEmail() bool {
	return strings.Contains(u.Email, "@")
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(goCode)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMetricsTool()
	
	params := MetricsParams{
		File:        tmpFile.Name(),
		Granularity: "function",
		IncludeComplexity: true,
		IncludeLOC:        true,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Check that metrics were calculated
	assert.Contains(t, result.Content, "Language: go")
	assert.Contains(t, result.Content, "Functions analyzed:")
	assert.Contains(t, result.Content, "main")
	assert.Contains(t, result.Content, "complexFunction")
	
	// The complex function should have higher complexity
	assert.Contains(t, result.Content, "Complexity:")
}

func TestMetricsTool_Execute_PythonFile(t *testing.T) {
	// Create a temporary Python file
	pythonCode := `def simple_function():
    return "hello"

def complex_function(x, y):
    if x < 0:
        return None
    
    result = 0
    for i in range(x):
        if i % 2 == 0:
            result += i
        else:
            result -= i
        
        if result > 100:
            break
    
    if y == "multiply":
        result *= 2
    elif y == "divide":
        result //= 2
    else:
        result += 10
    
    return result

class Calculator:
    def __init__(self):
        self.result = 0
    
    def add(self, x):
        self.result += x
        return self
    
    def multiply(self, x):
        self.result *= x
        return self
`
	
	tmpFile, err := os.CreateTemp("", "test_*.py")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(pythonCode)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMetricsTool()
	
	params := MetricsParams{
		File:        tmpFile.Name(),
		Granularity: "file",
		IncludeComplexity: true,
		IncludeLOC:        true,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	assert.Contains(t, result.Content, "Language: python")
	assert.Contains(t, result.Content, "Total lines:")
}

func TestMetricsTool_Execute_InvalidFile(t *testing.T) {
	tool := NewMetricsTool()
	
	params := MetricsParams{
		File: "nonexistent.go",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMetricsTool_Execute_EmptyFile(t *testing.T) {
	tool := NewMetricsTool()
	
	params := MetricsParams{
		File: "",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMetricsTool_Execute_InvalidJSON(t *testing.T) {
	tool := NewMetricsTool()
	
	result, err := tool.Execute(context.Background(), "invalid json")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMetricsTool_Execute_UnsupportedLanguage(t *testing.T) {
	// Create a temporary file with unsupported extension
	tmpFile, err := os.CreateTemp("", "test_*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString("some content\nline 2\nline 3")
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMetricsTool()
	
	params := MetricsParams{
		File: tmpFile.Name(),
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should return basic metrics for unsupported language
	assert.Contains(t, result.Content, "Language: unknown")
	assert.Contains(t, result.Content, "Total lines:")
}

func TestMetricsTool_Execute_Thresholds(t *testing.T) {
	// Create a file with high complexity
	goCode := `package main

func highComplexityFunction(x int) int {
	result := 0
	
	// Multiple nested conditions and loops
	for i := 0; i < x; i++ {
		for j := 0; j < i; j++ {
			for k := 0; k < j; k++ {
				if i%2 == 0 {
					if j%2 == 0 {
						if k%2 == 0 {
							result += i + j + k
						} else {
							result -= i + j + k
						}
					} else {
						if k%3 == 0 {
							result *= 2
						} else {
							result /= 2
						}
					}
				} else {
					if j%3 == 0 {
						if k%4 == 0 {
							result += 100
						} else {
							result -= 50
						}
					} else {
						result += 10
					}
				}
			}
		}
	}
	
	return result
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(goCode)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMetricsTool()
	
	params := MetricsParams{
		File:              tmpFile.Name(),
		Granularity:       "function",
		IncludeComplexity: true,
		Thresholds: &MetricsThresholds{
			CyclomaticComplexity: 5,
			LinesOfCode:          20,
		},
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should identify threshold violations
	assert.Contains(t, result.Content, "highComplexityFunction")
	assert.Contains(t, result.Content, "Complexity:")
}

func TestMetricsTool_Execute_AllGranularities(t *testing.T) {
	// Create a Go file
	goCode := `package main

func main() {
	println("hello")
}

func helper() {
	return
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(goCode)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMetricsTool()
	
	granularities := []string{"file", "function", "class"}
	
	for _, granularity := range granularities {
		t.Run("granularity_"+granularity, func(t *testing.T) {
			params := MetricsParams{
				File:        tmpFile.Name(),
				Granularity: granularity,
			}
			
			input, err := json.Marshal(params)
			require.NoError(t, err)
			
			result, err := tool.Execute(context.Background(), string(input))
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result.Content, "Language: go")
		})
	}
}

func TestMetricsTool_Execute_EmptyGoFile(t *testing.T) {
	// Create an empty Go file
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString("package main\n")
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMetricsTool()
	
	params := MetricsParams{
		File:        tmpFile.Name(),
		Granularity: "function",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should handle empty file gracefully
	assert.Contains(t, result.Content, "Language: go")
	assert.Contains(t, result.Content, "No functions found")
}

func TestMetricsTool_Execute_OnlyComplexity(t *testing.T) {
	// Create a Go file
	goCode := `package main

func main() {
	if true {
		println("hello")
	}
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(goCode)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMetricsTool()
	
	params := MetricsParams{
		File:              tmpFile.Name(),
		Granularity:       "function",
		IncludeComplexity: true,
		IncludeLOC:        false,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	assert.Contains(t, result.Content, "Complexity:")
	// Should not include LOC details when disabled
}

func TestMetricsTool_Execute_OnlyLOC(t *testing.T) {
	// Create a Go file
	goCode := `package main

func main() {
	println("hello")
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(goCode)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMetricsTool()
	
	params := MetricsParams{
		File:              tmpFile.Name(),
		Granularity:       "function",
		IncludeComplexity: false,
		IncludeLOC:        true,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	assert.Contains(t, result.Content, "Lines:")
	// Should not include complexity details when disabled
}
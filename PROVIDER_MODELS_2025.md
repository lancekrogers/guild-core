# Guild Provider Models - 2025 Update

This document summarizes the comprehensive update to Guild's provider system with the latest available models as of 2025.

## ✅ Updated Providers

### 🤖 OpenAI Provider
**Latest Models Added:**
- **GPT-4.1 Series** (Released April 2025) - Latest flagship models
  - `gpt-4.1` - Main model ($2/$8 per million tokens)
  - `gpt-4.1-mini` - Efficient variant ($0.4/$1.6 per million tokens)  
  - `gpt-4.1-nano` - Ultra cost-efficient ($0.1/$0.4 per million tokens)
- **GPT-4o Series** - Multimodal capabilities
  - `gpt-4o` - Full multimodal model
  - `gpt-4o-mini` - Efficient multimodal
- **o1/o3 Series** - Advanced reasoning models
  - `o1`, `o1-mini` - Original reasoning models
  - `o3-mini`, `o4-mini` - Latest reasoning variants
- **Audio Models** - Real-time audio processing
  - `gpt-4o-mini-realtime-preview`

**Key Features:**
- Up to 1 million token context windows
- Advanced coding capabilities (54.6% on SWE-bench vs 33.2% for GPT-4o)
- Model validation and intelligent defaults
- Cost tracking and usage recommendations

### 🧠 Anthropic Provider
**Latest Models Added:**
- **Claude 4 Series** (Released May 2025) - Latest generation
  - `claude-4-opus` - Most intelligent model ($15/$75 per million tokens)
  - `claude-4-sonnet` - Balanced performance ($3/$15 per million tokens)
- **Claude 3.7 Series** (Released February 2025)
  - `claude-3.7-sonnet` - Hybrid reasoning model
- **Claude 3 Series** - Previous generation (still available)
  - `claude-3-5-sonnet-20241022`, `claude-3-5-haiku-20241022`, etc.

**Key Features:**
- 200k token context windows across all models
- Best-in-class reasoning capabilities
- Hybrid reasoning with visible step-by-step thinking
- Enhanced safety measures

### 🔍 Google Provider
**Latest Models Added:**
- **Gemini 2.5 Series** (Released 2025) - Latest flagship
  - `gemini-2.5-pro` - Most intelligent model
  - `gemini-2.5-flash` - Efficient workhorse ($0.075/$0.3 per million tokens)
  - `gemini-2.5-pro-deep` - Deep thinking capabilities
- **Gemini 2.0 Series**
  - `gemini-2.0-flash` - Next-gen multimodal
  - `gemini-2.0-flash-lite` - Optimized for speed
- **Audio Models**
  - `gemini-2.5-flash-audio` - Native audio dialog

**Key Features:**
- Up to 2 million token context windows
- Leading multimodal capabilities
- Live API for real-time conversations
- Music generation capabilities (Lyria RealTime)

### 🦙 Ollama Provider
**Latest Models Added:**
- **Latest Flagship Models** (2025)
  - `llama3.3:70b` - Meta's latest 70B model
  - `qwen3:72b` - Latest Qwen generation
  - `deepseek-r1:70b` - Advanced reasoning model
  - `phi4:14b` - Microsoft's latest efficient model
- **Vision/Multimodal**
  - `llama3.2-vision:11b`, `qwen2-vl:7b`
- **Specialized Models**
  - `qwen2-math:7b` - Math specialized
  - `codegemma:7b` - Code specialized
  - `phi4-mini:3.8b` - Small efficient model

**Key Features:**
- RAM-based model recommendations
- Local deployment with no API costs
- Specialized models for different use cases
- Support for 2GB to 64GB+ RAM configurations

## 🎯 Enhanced Features

### Model Validation & Defaults
- Automatic model validation with intelligent fallbacks
- Latest models set as defaults (e.g., `gpt-4.1`, `claude-4-sonnet`)
- Model information including pricing and capabilities

### Smart Recommendations
- Use case-based model recommendations:
  - **Coding**: `gpt-4.1`, `claude-4-opus`, `codegemma:7b`
  - **Reasoning**: `o3-mini`, `claude-4-opus`, `deepseek-r1:70b`
  - **Cost-efficient**: `gpt-4.1-nano`, `claude-3-5-haiku`, `gemini-2.5-flash`
  - **Multimodal**: `gpt-4o`, `gemini-2.5-pro`, `llama3.2-vision:11b`

### Registry Integration
- Seamless integration with Guild's ComponentRegistry
- Configuration-driven provider setup
- Environment variable support for API keys
- Dynamic provider switching

## 📊 Model Comparison (2025)

| Provider | Model | Type | Context | Input Price | Output Price | Strengths |
|----------|-------|------|---------|-------------|--------------|-----------|
| OpenAI | gpt-4.1 | text | 1M tokens | $2.00 | $8.00 | Best coding, 1M context |
| Anthropic | claude-4-opus | reasoning | 200k tokens | $15.00 | $75.00 | Best reasoning, complex tasks |
| Google | gemini-2.5-pro | multimodal | 2M tokens | $3.50 | $10.50 | Multimodal, large context |
| Ollama | llama3.3:70b | text | Local | Free | Free | Local, no API costs |
| OpenAI | gpt-4.1-nano | text | 1M tokens | $0.10 | $0.40 | Ultra cost-efficient |
| Google | gemini-2.5-flash | multimodal | 1M tokens | $0.075 | $0.30 | Fast and cheap |

## 🚀 Usage Examples

### Registry-Based Usage
```go
// Create registry with latest models
registry := NewComponentRegistry()
config := LoadConfig("config.yaml") // Contains latest model configs
registry.Initialize(ctx, *config)

// Use latest models through registry
provider, _ := registry.Providers().GetDefaultProvider()
response, _ := provider.Complete(ctx, "Hello world!")
```

### Direct Provider Usage
```go
// Latest OpenAI model
client := openai.NewClient("sk-...", "gpt-4.1")
response, _ := client.Complete(ctx, "Write some code")

// Latest Claude model  
client := anthropic.NewClient("sk-ant-...", "claude-4-sonnet")
response, _ := client.Complete(ctx, "Explain reasoning")

// Latest Gemini model
client := google.NewClient("...", "gemini-2.5-flash")
response, _ := client.Complete(ctx, "Analyze this image")
```

### Smart Model Selection
```go
// Get recommendations based on use case
codingModel := openai.GetRecommendedModel("coding")        // gpt-4.1
reasoningModel := anthropic.GetRecommendedModel("reasoning") // claude-4-opus
cheapModel := google.GetRecommendedModel("cost-efficient")  // gemini-2.5-flash

// Ollama with RAM constraints
localModel := ollama.GetRecommendedModel("coding", 8) // codegemma:7b
```

## 🔧 Configuration Updates

Default configurations now use the latest models:

```yaml
providers:
  default_provider: "openai"
  providers:
    openai:
      model: "gpt-4.1"            # Updated from gpt-4
      api_key_env: "OPENAI_API_KEY"
    anthropic:
      model: "claude-4-sonnet"     # Updated from claude-3-sonnet
      api_key_env: "ANTHROPIC_API_KEY"
    google:                        # New provider added
      model: "gemini-2.5-flash"
      api_key_env: "GOOGLE_API_KEY"
    ollama:
      model: "llama3.1:8b"         # Updated from llama2
      url: "http://localhost:11434"
```

## ✅ Testing

Comprehensive test coverage includes:
- Model validation and defaults for all providers
- Smart recommendation algorithms
- Registry integration with latest models
- Factory pattern with 2025 models
- Error handling and fallbacks

## 🎉 Benefits

1. **Current Models**: Access to the very latest AI models (2025)
2. **Cost Optimization**: Smart recommendations for cost-efficient choices
3. **Performance**: Latest models with improved capabilities
4. **Flexibility**: Easy switching between providers and models
5. **Local Options**: Comprehensive Ollama support for privacy/cost
6. **Future-Proof**: Easy to add new models as they're released

This update ensures Guild users have access to the most current and capable AI models available, with intelligent defaults and recommendations for different use cases.
package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/providers"
)

func newTestProviderService(t *testing.T, cfg *config.GuildConfig) *ProviderService {
	ps, err := NewProviderService(context.Background(), cfg)
	require.NoError(t, err)
	ps.providerStatus["openai"] = ProviderStatus{Name: "openai", Type: "openai"}
	ps.providerStatus["anthropic"] = ProviderStatus{Name: "anthropic", Type: "anthropic"}
	ps.providerStatus["ollama"] = ProviderStatus{Name: "ollama", Type: "ollama"}
	return ps
}

func TestProviderService_CheckOpenAIHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"object":"list","data":[]}`))
	}))
	defer server.Close()

	cfg := &config.GuildConfig{}
	cfg.Providers.OpenAI.BaseURL = server.URL
	ps := newTestProviderService(t, cfg)
	os.Setenv(providers.EnvOpenAIKey, "test")
	defer os.Unsetenv(providers.EnvOpenAIKey)

	ok, err := ps.checkOpenAIHealth("openai")
	assert.NoError(t, err)
	assert.True(t, ok)

	serverFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer serverFail.Close()
	cfg.Providers.OpenAI.BaseURL = serverFail.URL

	ok, err = ps.checkOpenAIHealth("openai")
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestProviderService_CheckAnthropicHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"object":"list","data":[]}`))
	}))
	defer server.Close()

	cfg := &config.GuildConfig{}
	cfg.Providers.Anthropic.BaseURL = server.URL
	ps := newTestProviderService(t, cfg)
	os.Setenv(providers.EnvAnthropicKey, "ant-test")
	defer os.Unsetenv(providers.EnvAnthropicKey)

	ok, err := ps.checkAnthropicHealth("anthropic")
	assert.NoError(t, err)
	assert.True(t, ok)

	serverFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer serverFail.Close()
	cfg.Providers.Anthropic.BaseURL = serverFail.URL

	ok, err = ps.checkAnthropicHealth("anthropic")
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestProviderService_CheckOllamaHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"models":[]}`))
	}))
	defer server.Close()

	cfg := &config.GuildConfig{}
	cfg.Providers.Ollama.BaseURL = server.URL
	ps := newTestProviderService(t, cfg)

	ok, err := ps.checkOllamaHealth("ollama")
	assert.NoError(t, err)
	assert.True(t, ok)

	serverFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer serverFail.Close()
	cfg.Providers.Ollama.BaseURL = serverFail.URL

	ok, err = ps.checkOllamaHealth("ollama")
	assert.Error(t, err)
	assert.False(t, ok)
}

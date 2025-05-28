# Google Provider Removal

The Google provider has been temporarily removed from the Guild Framework as it needs to be updated to implement the new `AIProvider` interface. This is a low priority task that will be addressed after MVP.

## What was removed:
- `/pkg/providers/google/` directory and all its contents
- Google provider references from:
  - `pkg/providers/factory.go`
  - `pkg/providers/factory_v2.go`
  - `pkg/providers/example_2025_models.go`
  - `Makefile` provider lists

## Current status:
- All other providers (OpenAI, Anthropic, DeepSeek, DeepInfra, Ollama, Ora, Mock) are fully functional
- All provider tests passing
- Build succeeds without Google provider

## To re-add Google provider (post-MVP):
1. Create new `/pkg/providers/google/` package
2. Implement the `interfaces.AIProvider` interface
3. Add back to factory files
4. Update provider lists in Makefile
5. Add comprehensive tests using the provider testing framework

## Note
The removal was done to unblock development as the Google provider was using an outdated interface and preventing the build from succeeding after the major provider refactoring.
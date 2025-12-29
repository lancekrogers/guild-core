#!/usr/bin/env just --justfile
# guild-core build and development tasks

set dotenv-load := true

# Configuration
binary_name := "guild"
bin_dir := "bin"
gobin := env_var_or_default("GOBIN", `go env GOPATH` + "/bin")
BUILDTOOL := "go run ./internal/buildutil"
export GOCACHE := `pwd` + "/.cache/go-build"

# Modules
[doc('Cross-platform builds')]
mod xbuild '.justfiles/build.just'

[doc('Testing (unit, integration, e2e, happy path)')]
mod test '.justfiles/test.just'

[doc('Docker workflows')]
mod docker '.justfiles/docker.just'

[doc('Release packaging')]
mod release '.justfiles/release.just'

[private]
default:
    #!/usr/bin/env bash
    echo "guild-core"
    echo ""
    just --list --unsorted

# Build guild binary with visual dashboard
build:
    @{{BUILDTOOL}} build

# Build guild binary only (fast, no vet)
build-only:
    @{{BUILDTOOL}} build-only

# Format Go code
fmt:
    go fmt ./...
    gofumpt -w .

# Run go vet
vet:
    go vet ./...

# Run formatting and vetting
lint: fmt vet
    @echo "✅ Linting complete"

# Clean build artifacts with visual dashboard
clean:
    @{{BUILDTOOL}} clean

# Update and tidy dependencies
deps:
    go get -u ./...
    go mod tidy

# Install guild to $GOBIN
install: build-only
    @{{BUILDTOOL}} install

# Uninstall guild from $GOBIN
uninstall:
    @{{BUILDTOOL}} uninstall

# Show project dashboard
dashboard:
    @{{BUILDTOOL}} dashboard

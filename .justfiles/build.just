# Build recipes
binary_name := "guild"
main_pkg := "./cmd/guild"
bin_dir := "bin"

# Show available build commands
[private]
default:
    @just --list --unsorted --justfile {{source_file()}}

# Quick development build (no optimization)
[no-cd]
dev:
    mkdir -p {{bin_dir}} && go build -o {{bin_dir}}/{{binary_name}} {{main_pkg}}

# Build for Linux AMD64
[no-cd]
linux:
    mkdir -p {{bin_dir}} && GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' -o {{bin_dir}}/{{binary_name}}-linux-amd64 {{main_pkg}}

# Build for macOS Intel
[no-cd]
darwin:
    mkdir -p {{bin_dir}} && GOOS=darwin GOARCH=amd64 go build -ldflags '-s -w' -o {{bin_dir}}/{{binary_name}}-darwin-amd64 {{main_pkg}}

# Build for macOS Apple Silicon
[no-cd]
darwin-arm64:
    mkdir -p {{bin_dir}} && GOOS=darwin GOARCH=arm64 go build -ldflags '-s -w' -o {{bin_dir}}/{{binary_name}}-darwin-arm64 {{main_pkg}}

# Build for Windows AMD64
[no-cd]
windows:
    mkdir -p {{bin_dir}} && GOOS=windows GOARCH=amd64 go build -ldflags '-s -w' -o {{bin_dir}}/{{binary_name}}-windows-amd64.exe {{main_pkg}}

# Build for all platforms
all: linux darwin darwin-arm64 windows
    @echo "✅ All platform builds complete"

# Build Linux binary for container testing
[no-cd]
test-binary:
    mkdir -p {{bin_dir}}/linux && GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' -o {{bin_dir}}/linux/{{binary_name}} {{main_pkg}}

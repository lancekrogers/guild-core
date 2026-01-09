# Callgraph recipes
module_path := "github.com/guild-framework/guild-core"
out_dir := "docs/images/callgraphs"
callvis := "go-callvis"
go_cache := "$PWD/.cache/go-build"

# Show available callgraph commands
[private]
default:
    @just --list --unsorted --justfile {{source_file()}}

# Ensure output directory exists
[no-cd]
setup:
    mkdir -p {{out_dir}} {{go_cache}}

# Interactive callgraph server for the Great Hall CLI
[no-cd]
server-cli:
    {{callvis}} -nostd -group pkg -focus {{module_path}}/cmd/guild ./cmd/guild

# CLI -> daemon integration overview
[no-cd]
cli-daemon: setup
    {{callvis}} -nostd -nointer -group pkg -graphviz -format svg -file {{out_dir}}/cli-daemon -focus {{module_path}}/cmd/guild -limit {{module_path}}/cmd/guild,{{module_path}}/internal/daemon,{{module_path}}/internal/daemonconn,{{module_path}}/internal/integration ./cmd/guild

# Daemon lifecycle callgraph
[no-cd]
daemon-lifecycle: setup
    {{callvis}} -nostd -group pkg -graphviz -format svg -file {{out_dir}}/daemon-lifecycle -focus {{module_path}}/internal/daemon -limit {{module_path}}/internal/daemon,{{module_path}}/internal/integration,{{module_path}}/pkg ./internal/daemon

# Integration services overview
[no-cd]
integration-services: setup
    {{callvis}} -nostd -group pkg -graphviz -format svg -file {{out_dir}}/integration-services -focus {{module_path}}/internal/integration/services -limit {{module_path}}/internal/integration,{{module_path}}/pkg ./internal/integration/services

# Bulk callgraphs for non-UI packages
[no-cd]
non-ui: setup
    @packages=$(rg --files -g '*.go' internal pkg | rg -v '_test\\.go$' | rg -v '^internal/ui/' | sed 's#/[^/]*$##' | sort -u); \
    for pkg in $packages; do \
        out_name=$(echo $pkg | sed 's#/#-#g'); \
        GOCACHE={{go_cache}} {{callvis}} -nostd -group pkg -graphviz -format svg -file {{out_dir}}/$out_name -focus {{module_path}}/$pkg -limit {{module_path}} ./$pkg; \
    done

# Generate all standard callgraphs
[no-cd]
all: cli-daemon daemon-lifecycle integration-services

# Generate static HTML viewer for callgraphs
[no-cd]
viewer:
    @echo "Generating callgraph viewer..."
    go run ./tools/callgraph-viewer
    @echo "✓ Viewer generated at docs/callgraphs/viewer/index.html"

# Generate viewer and start server at localhost:9000
[no-cd]
view: viewer
    @echo "✓ Viewer ready at http://localhost:9000"
    @echo "Press Ctrl+C to stop the server"
    @echo ""
    cd docs/callgraphs/viewer && python3 -m http.server 9000

# Regenerate callgraphs and viewer
[no-cd]
refresh: non-ui viewer

# Serve viewer with local HTTP server (required for ES modules)
[no-cd]
serve-viewer:
    @echo "Starting viewer at http://localhost:9000"
    @echo "Press Ctrl+C to stop"
    cd docs/callgraphs/viewer && python3 -m http.server 9000

# Open viewer in default browser (deprecated - use serve-viewer instead)
[no-cd]
open-viewer:
    @echo "Warning: Opening file:// may not work due to CORS restrictions"
    @echo "Use 'just callgraph serve-viewer' instead"
    open docs/callgraphs/viewer/index.html

# Watch for SVG changes and regenerate viewer
[no-cd]
viewer-watch:
    watchexec -w docs/images/callgraphs -e svg -- just callgraph viewer

# Clean generated viewer files
[no-cd]
clean-viewer:
    rm -rf docs/callgraphs/viewer

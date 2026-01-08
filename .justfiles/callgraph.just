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

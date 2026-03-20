# Guild CLI Tab Completion

The Guild CLI supports intelligent tab completion for commands, subcommands, flags, and dynamic values like campaign IDs and agent names.

## Quick Start

### Install Completion (Automatic)

```bash
make install  # Installs Guild and auto-detects your shell for completion
```

### Install Completion (Manual)

#### Bash

```bash
# One-time setup
guild completion bash > /etc/bash_completion.d/guild

# Or for macOS with Homebrew:
guild completion bash > $(brew --prefix)/etc/bash_completion.d/guild

# Or for current session only:
source <(guild completion bash)
```

#### Zsh

```bash
# Enable completion if not already done:
echo "autoload -U compinit; compinit" >> ~/.zshrc

# Install Guild completion:
guild completion zsh > "${fpath[1]}/_guild"

# Restart your shell
```

#### Fish

```bash
guild completion fish > ~/.config/fish/completions/guild.fish

# Or for current session only:
guild completion fish | source
```

#### PowerShell

```powershell
# For current session:
guild completion powershell | Out-String | Invoke-Expression

# For all sessions:
guild completion powershell > guild.ps1
# Add to your PowerShell profile
```

## Features

### Command Completion

Press `<TAB>` to see available commands:

```bash
$ guild <TAB>
agent       campaign    chat        commission  completion  corpus
init        kanban      migrate     prompt      serve       version
```

### Subcommand Completion

```bash
$ guild campaign <TAB>
create    list      start     status    watch

$ guild agent <TAB>
start
```

### Flag Completion

```bash
$ guild chat --<TAB>
--campaign    --session     --help

$ guild campaign create --<TAB>
--commission    --manager    --name    --help
```

### Dynamic Value Completion

#### Campaign IDs

When using `--id` flags with campaign commands:

```bash
$ guild campaign start --id <TAB>
campaign-1734567890    e-commerce campaign
campaign-1734567900    performance optimization
```

#### Agent IDs

For agent selection:

```bash
$ guild agent start <TAB>
manager     Project management, task decomposition, coordination...
backend     Backend development, API design, database...
frontend    Frontend development, UI/UX, React...

$ guild campaign create --manager <TAB>
manager     Project management, task decomposition, coordination...
devops      Infrastructure, deployment, CI/CD...
```

#### Commission Files

For commission file paths:

```bash
$ guild campaign create --commission <TAB>
.guild/objectives/api-design.md           api design
.guild/objectives/refined/e-commerce.md   Refined: e commerce
```

#### Campaign Names

For campaign selection in chat:

```bash
$ guild chat --campaign <TAB>
e-commerce          Status: active
api-development     Status: ready
frontend-redesign   Status: completed
```

## Troubleshooting

### Completions Not Working

1. **Start a new shell session** after installation
2. **Verify installation location:**

   ```bash
   # Bash
   ls /etc/bash_completion.d/guild
   
   # Zsh
   echo $fpath | grep -o '[^ ]*' | xargs ls | grep _guild
   
   # Fish
   ls ~/.config/fish/completions/guild.fish
   ```

3. **Check if completion is loaded:**

   ```bash
   # Bash
   complete | grep guild
   
   # Zsh
   print -l ${(ok)_comps} | grep guild
   ```

### Dynamic Completions Show Errors

1. **Ensure Guild is initialized:**

   ```bash
   guild init
   ```

2. **Check project context:**

   ```bash
   # Must be in a Guild project directory
   ls .guild/
   ```

3. **Verify database access:**

   ```bash
   # SQLite database should exist
   ls .guild/memory.db
   ```

### Permission Denied

For system-wide installation, use sudo:

```bash
sudo guild completion bash > /etc/bash_completion.d/guild
```

Or install for current user only:

```bash
# Bash
mkdir -p ~/.local/share/bash-completion/completions
guild completion bash > ~/.local/share/bash-completion/completions/guild

# Zsh
mkdir -p ~/.zsh/completions
guild completion zsh > ~/.zsh/completions/_guild
# Add to ~/.zshrc: fpath=(~/.zsh/completions $fpath)
```

## Advanced Usage

### Debugging Completions

Enable debug output:

```bash
# Bash
export BASH_COMP_DEBUG_FILE=/tmp/guild-completion.log

# Then use tab completion and check the log
cat /tmp/guild-completion.log
```

### Custom Completions

The completion system is extensible. To add custom completions for new commands:

1. Add a completion function in `cmd/guild/completions.go`
2. Register it with the command using `RegisterFlagCompletionFunc` or `ValidArgsFunction`
3. Rebuild and reinstall completions

Example:

```go
// In your command's init() function
myCmd.RegisterFlagCompletionFunc("my-flag", completeMyCustomValues)
```

## Tips

1. **Double-tab** shows all available completions
2. **Type partial text** then tab to filter completions
3. **Escape spaces** in paths with backslash or quotes
4. **Update completions** after Guild updates: `make install-completion`

## Shell-Specific Notes

### Bash

- Requires bash-completion package
- On macOS, install via: `brew install bash-completion`

### Zsh

- Built-in completion support
- More advanced completion features than bash

### Fish

- Automatic completion loading from `~/.config/fish/completions/`
- Rich descriptions shown by default

### PowerShell

- Works on Windows, macOS, and Linux
- Requires PowerShell 5.0 or later

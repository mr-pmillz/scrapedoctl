# 02 - Installation & Setup

## Prerequisites

- **Go 1.26+** (for building from source)
- **Podman/Docker** (optional, for containerized development)
- **Scrape.do API Token** (available at [scrape.do](https://scrape.do/))

## Building from Source

To build the `scrapedoctl` binary locally:

```bash
# Clone the repository
git clone https://github.com/mr-pmillz/scrapedoctl.git
cd scrapedoctl

# Build the binary
go build -o bin/scrapedoctl ./cmd/scrapedoctl
```

## Interactive Installation

`scrapedoctl` features a built-in interactive installer that sets up your configuration file and automatically integrates with your AI agents.

To trigger the installer, simply run any command without a configuration file:

```bash
./bin/scrapedoctl scrape https://example.com
```

## Shell Completion

`scrapedoctl` supports automatic shell completion for Bash, Zsh, Fish, and PowerShell. You can either generate scripts manually or use the built-in `completion install` command for automatic XDG-compliant installation.

### Automatic Installation (Recommended)

The `completion install` subcommand writes completion scripts to the standard XDG/system directories. No `.bashrc` or `.zshrc` editing is required -- the shell's native completion system picks up the files automatically.

```bash
# Install for your shell (bash, zsh, or fish)
scrapedoctl completion install bash
scrapedoctl completion install zsh
scrapedoctl completion install fish
```

For Bash, the script is placed in `$XDG_DATA_HOME/bash-completion/completions/` (user) or `/usr/share/bash-completion/completions/` (system). Zsh and Fish follow similar conventions. Restart your shell after installing.

### Manual Generation

If you prefer to source completions directly, you can generate the script to stdout:

### Bash
Add the following to your `~/.bashrc`:
```bash
source <(scrapedoctl completion bash)
```

### Zsh
Add the following to your `~/.zshrc`:
```zsh
source <(scrapedoctl completion zsh)
```

### Oh My Zsh
If you are using [Oh My Zsh](https://ohmyz.sh/), you can create a custom completion file:
```bash
mkdir -p ~/.oh-my-zsh/completions
scrapedoctl completion zsh > ~/.oh-my-zsh/completions/_scrapedoctl
```
Then restart your shell or run `source ~/.zshrc`.

### PowerShell
`scrapedoctl` provides a native PowerShell module for command completion, compatible with PowerShell 7.6+ on Windows, Linux, and macOS.

#### Installation
1. Generate the module and manifest:
   ```powershell
   scrapedoctl completion powershell > scrapedoctl.psm1
   # The release also includes a pre-generated scrapedoctl.psd1 manifest
   ```
2. Import the module:
   ```powershell
   Import-Module ./scrapedoctl.psm1
   ```
3. To make it persistent, add the import command to your `$PROFILE`.

#### Features for PowerShell 7.4+
- Supports `NativeCommandErrorActionPreference` for better error handling.
- Optimized for cross-platform usage on Unix-based systems.

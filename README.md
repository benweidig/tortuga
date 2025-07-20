![CI Workflow](https://github.com/benweidig/tortuga/actions/workflows/ci.yml/badge.svg)

# Tortuga 🐢

**Fast, parallel Git operations across multiple repositories**

Tortuga is a modern CLI tool for managing multiple Git repositories simultaneously.
It can fetch, rebase, and push changes across all your repositories with a single command, providing real-time progress updates.

![Tortuga Mascot](mascot.png)  

## ✨ Features

*   **🚀 Parallel Operations**:
     Process multiple repositories concurrently for maximum speed

*   **📊 Real-time Progress**:
    Live updates showing the status of each repository

*   **🔄 Smart Sync Options**:
    Choose between full sync or incoming-only updates

*   **🎨 Beautiful Output**:
    Color-coded status with clear visual indicators

*   **⚡ Thread-safe**:
    Modern Go concurrency patterns with channels and errgroup

*   **🛡️ Error Resilient**:
    Continue processing even if some repositories fail

*   **🌈 Color Support**:
    Respects terminal capabilities and `NO_COLOR` standard

---

## 🚀 Quick Start

### Installation

**macOS (Homebrew):**

```bash
brew install benweidig/homebrew-tap/tortuga
```

**From Source:**

```bash
go install github.com/benweidig/tortuga@latest
```

**Pre-built Binaries:**

Download from [releases](https://github.com/benweidig/tortuga/releases) or use the provided `.deb` packages for Debian/Ubuntu.


### Basic Usage

```bash
# Check all repositories in current directory
tt

# Check repositories in specific path
tt /path/to/my/projects

# Auto-sync all changes without prompting
tt --yes

# Use monochrome output (no colors)
tt --monochrome
```

---

## 📖 How It Works

1.  **Discovery**:
    Tortuga scans the target directory for Git repositories

2.  **Fetch**:
    Parallel fetch from all remotes to check for updates

3.  **Status Display**:
    Real-time status showing incoming/outgoing commits and local changes

4.  **Interactive Sync**: Choose your sync strategy:
    *   `y` - Full sync (stash → pull+rebase → push)
    *   `i` - Incoming only (stash → pull+rebase)
    *   `n` - No sync (just show status)

---

## ⚙️ Configuration

### Command Line Options

| Flag               | Short | Description                               |
| ------------------ | ----- | ----------------------------------------- |
| `--monochrome`     | `-m`  | Disable ANSI colors                       |
| `--yes`            | `-y`  | Automatically accept sync prompts         |
| `--verbose`        | `-v`  | Show verbose error output                 |

### Git Credentials

Tortuga performs operations asynchronously across multiple repositories, so it cannot prompt for credentials interactively.
Ensure your Git credentials are configured via:

*   **SSH Keys**:
    Recommended for seamless authentication

*   **Git Credential Helper**:
    `git config credential.helper store`

*   **Git Credential Cache**:
    `git config credential.helper cache`

### Color Support

Colors are automatically detected and disabled when:

*   Terminal doesn't support ANSI colors
*   `NO_COLOR` environment variable is set
*   `--monochrome` flag is used

---

## 🏗️ Architecture

Tortuga is built with modern Go patterns:

- **Channel-based rendering**: Thread-safe UI updates without mutexes
- **errgroup**: Proper error handling in concurrent operations  
- **Repository Manager**: Clean abstraction for batch operations
- **Git Interface**: Mockable Git operations for testing
- **Structured Types**: Type-safe data structures instead of raw parsing

---

## 🔧 Building from Source

### Requirements

Go 1.24 or later

### Build

```bash
git clone https://github.com/benweidig/tortuga.git
cd tortuga
make build
```

### Development

```bash
make all          # Full pipeline: clean, format, test, vet, staticcheck, build
make test         # Run tests
make fmt          # Format code
make vet          # Run go vet
make staticcheck  # Run staticcheck (if installed)
```

---

## 📄 License

MIT. See [LICENSE](LICENSE).

## 🙏 Acknowledgments

*   Mascot design [based on Gopherize.me](https://gopherize.me/gopher/79e06dc4b7a8669c8aa0d6381af7f02f5474e3b7)

*   Git logo by [Jason Long](https://git-scm.com/downloads/logos) under CC BY 3.0

*   Original UI inspiration from [gosuri/uilive](https://github.com/gosuri/uilive) (MIT License)

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

### Development Setup

1.  Fork the repository

2.  Create your feature branch (`git checkout -b feature/amazing-feature`)

3.  Run tests (`make test`)

4.  Commit your changes (`git commit -m 'Add amazing feature'`)

5.  Push to the branch (`git push origin feature/amazing-feature`)

6.  Open a Pull Request

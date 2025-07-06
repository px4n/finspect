# finspect

A unified filesystem interface for all your data sources - local files, cloud storage, and social media - accessible through a single, POSIX-like API.

## Vision

finspect treats all data sources as a unified filesystem. Whether your files are on your local drive, in Google Drive, or your photos are on Facebook, finspect lets you access and manage them through standard filesystem operations.

## Features

### Current (Phase 1)

- **Virtual Filesystem (VFS)**: Mount any data source and access it like a local filesystem
- **Local Filesystem Adaptor**: Full support for local files and directories
- **POSIX Operations**: Standard file operations (ls, cp, mv, rm, stat)
- **File Watching**: Real-time notifications for file changes
- **CLI Interface**: Familiar command-line tools for file management

### Planned (Phase 2+)

- **Cloud Storage Adaptors**: Google Drive, Dropbox, OneDrive, S3
- **Social Media Adaptors**: Facebook photos, X posts, Instagram
- **Content-Addressable Storage**: Deduplicated blob storage for efficient space usage
- **Metadata & Search**: Rich metadata with full-text search across all sources
- **Workflow Automation**: Event-driven automation based on file changes
- **Web UI**: Browser-based interface for visual file management

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/px4n/finspect.git
cd finspect

# Build the project
make build

# Run tests
make test
```

### Basic Usage

```bash
# Mount a local directory
./bin/finspect mount local ~/Documents /docs

# List files
./bin/finspect ls /docs
./bin/finspect ls -la /docs  # Show hidden files in long format

# Copy files
./bin/finspect cp /docs/report.pdf /docs/backup/report-2024.pdf

# Move/rename files
./bin/finspect mv /docs/old-name.txt /docs/new-name.txt

# Remove files
./bin/finspect rm /docs/temp.txt
./bin/finspect rm -rf /docs/old-folder

# Get file information
./bin/finspect stat /docs/important.doc
```

### Advanced Example

```bash
# Mount multiple sources
./bin/finspect mount local ~/Pictures /pics
./bin/finspect mount local ~/Projects /projects
./bin/finspect mount local /mnt/backup /backup

# Work across mounts
./bin/finspect cp /pics/vacation/*.jpg /backup/photos/2024/
./bin/finspect ls -lH /projects  # Human-readable sizes
```

## Architecture

finspect uses a modular architecture with clear separation of concerns:

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│     CLI     │     │   Web UI    │     │     API     │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                    │
       └───────────────────┴────────────────────┘
                           │
                    ┌──────▼──────┐
                    │     VFS     │
                    │   Router    │
                    └──────┬──────┘
                           │
        ┌─────────────┬────┴────┬─────────────┐
        │             │         │             │
   ┌────▼────┐  ┌────▼────┐  ┌─▼──┐  ┌──────▼──────┐
   │Filesystem│  │ Google  │  │ S3 │  │  Facebook   │
   │ Adaptor  │  │  Drive  │  │    │  │   Photos    │
   └──────────┘  └─────────┘  └────┘  └─────────────┘
```

## Development

### Prerequisites

- Go 1.22 or higher
- Make
- Git

### Building from Source

```bash
# Clone the repository
git clone https://github.com/px4n/finspect.git
cd finspect

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run with verbose output
./bin/finspect -v ls /
```

### Project Structure

```
finspect/
├── adaptors/          # Data source adaptors
│   └── filesystem/    # Local filesystem adaptor
├── cmd/
│   └── finspect/     # CLI application
├── pkg/
│   └── vfs/          # Virtual filesystem core
├── internal/
│   └── pathutil/     # Path manipulation utilities
├── docs/             # Documentation
└── test/             # Integration tests
```

## Contributing

I welcome contributions. Please see the [Contributing Guide](CONTRIBUTING.md) for details.

### Areas for Contribution

- New adaptors (cloud storage, databases, APIs)
- UI improvements (web interface, desktop app)
- Performance optimizations
- Documentation and examples
- Bug fixes and testing

## Roadmap

### Phase 1 ✅ (Complete)

- Core VFS implementation
- Local filesystem adaptor
- Basic CLI operations
- Path manipulation utilities

### Phase 2 🚧 (In Progress)

- Metadata system
- Search and indexing
- Cloud storage adaptors
- Configuration management

### Phase 3 📋 (Planned)

- Social media adaptors
- Content-addressable blob storage
- Workflow automation
- Web-based UI

### Phase 4 🔮 (Future)

- Mobile apps
- P2P synchronization
- AI-powered organization
- Plugin system

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [Perkeep](https://perkeep.org/) (formerly Camlistore)
- Built with [Cobra](https://github.com/spf13/cobra) for CLI
- Uses [fsnotify](https://github.com/fsnotify/fsnotify) for file watching

## Contact

- GitHub Issues: [github.com/px4n/finspect/issues](https://github.com/px4n/finspect/issues)

---

**Note**: This project is under active development. APIs and features may change.

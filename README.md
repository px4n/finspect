# finspect

A unified filesystem interface for all your data sources - local files, cloud storage, and social media - accessible through a single, POSIX-like API.

## Vision

finspect treats all data sources as a unified filesystem. Whether your files are on your local drive, in Google Drive, or your photos are on Facebook, finspect lets you access and manage them through standard filesystem operations.

## Features

### Current

- **Virtual Filesystem (VFS)**: Mount any data source and access it like a local filesystem
- **POSIX Operations**: Standard file operations (ls, cp, mv, rm, stat)
- **Metadata & Search**: Extract metadata and perform full-text search with Bleve
- **Cloud Storage**: S3 and Google Drive support with streaming operations
- **Configuration**: YAML/JSON config with environment variable overrides
- **File Watching**: Real-time notifications for file changes

### Planned

- **More Cloud Storage**: Dropbox, OneDrive, Azure Blob Storage
- **Social Media Adaptors**: Facebook photos, X posts, Instagram
- **Content-Addressable Storage**: Deduplicated blob storage
- **Workflow Automation**: Event-driven file processing
- **Web UI**: Browser-based file management

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

### Metadata and Search

```bash
# Extract and store metadata for a file
./bin/finspect metadata extract /docs/report.pdf

# Index an entire directory for search
./bin/finspect metadata index /docs --full-text

# Search metadata by content type
./bin/finspect metadata search --type "application/pdf"

# Full-text search across indexed files
./bin/finspect metadata search --text "quarterly report" --full-text

# Show stored metadata for a file
./bin/finspect metadata show /docs/report.pdf
```

### S3 Integration

```bash
# Mount S3 bucket (requires AWS credentials)
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
./bin/finspect mount s3 my-bucket /s3 --region us-east-1

# List S3 files
./bin/finspect ls /s3

# Copy between local and S3
./bin/finspect cp /docs/report.pdf /s3/reports/2024/
./bin/finspect cp /s3/data.csv /docs/downloads/
```

### Configuration

```bash
# Initialize configuration file
./bin/finspect config init

# Show current configuration
./bin/finspect config show

# Use custom configuration
./bin/finspect --config custom.yaml ls /

# Configuration via environment variables
export FINSPECT_METADATA_DATABASE=/var/lib/finspect/metadata.db
export FINSPECT_SEARCH_INDEX=/var/lib/finspect/search.bleve
```

### Advanced Example

```bash
# Create configuration for multiple mounts
cat > finspect.yaml << EOF
mounts:
  - path: "/"
    type: "filesystem"
    config:
      root: "$HOME"
  - path: "/s3"
    type: "s3"
    config:
      bucket: "my-data"
      region: "us-east-1"
  - path: "/gdrive"
    type: "googledrive"
    config:
      credentials:
        service_account_key: "/path/to/key.json"
EOF

# Index everything with metadata and search
./bin/finspect metadata index / --full-text
./bin/finspect metadata index /s3 --full-text
./bin/finspect metadata index /gdrive --full-text

# Search across all mounted sources
./bin/finspect metadata search --text "project report" --full-text

# Copy files between different storage backends
./bin/finspect cp /gdrive/Documents/report.pdf /s3/backups/
./bin/finspect cp /s3/data/*.csv /local/analysis/
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
│   ├── filesystem/    # Local filesystem adaptor
│   ├── s3/           # AWS S3 adaptor
│   └── cloud/        # Cloud storage framework
├── cmd/
│   └── finspect/     # CLI application
├── pkg/
│   ├── vfs/          # Virtual filesystem core
│   ├── metadata/     # Metadata extraction and storage
│   └── search/       # Full-text search with Bleve
├── internal/
│   └── pathutil/     # Path manipulation utilities
├── examples/         # Usage examples
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

### Phase 1 (Complete)

- Core VFS implementation
- Local filesystem adaptor
- Basic CLI operations
- Path manipulation utilities

### Phase 2 (Complete)

- Metadata system with SQLite backend
- Full-text search and indexing with Bleve
- S3 cloud storage adaptor
- Cloud storage framework for extensibility

### Phase 3 (Planned)

- Social media adaptors
- Content-addressable blob storage
- Workflow automation
- Web-based UI

### Phase 4 (Future)

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

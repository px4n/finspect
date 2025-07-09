# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### Core Features
- Initial VFS (Virtual File System) implementation
- Local filesystem adaptor with full POSIX operations
- CLI with basic commands (mount, ls, cp, mv, rm, stat)
- File watching capabilities using fsnotify
- Path manipulation utilities

#### Metadata and Search
- Metadata extraction and storage system with SQLite backend
- Full-text search using Bleve search engine
- Metadata CLI commands (extract, index, search, show)
- Build tags for conditional SQLite FTS5 support

#### Cloud Storage
- S3 cloud storage adaptor with streaming support
- Google Drive adaptor with OAuth2 and service account auth
- Cloud storage framework for extensibility

#### Configuration
- Configuration management with YAML/JSON support
- Configuration CLI commands (init, show, validate, path)
- Environment variable configuration overrides

#### Development
- Comprehensive test suite
- GitHub Actions CI/CD pipeline
- Project documentation and contributing guidelines

### Changed
- Complete rewrite from the original Perkeep-based implementation
- Simplified architecture focusing on VFS abstraction
- Modular adaptor system for future extensibility
- Enhanced CLI with additional commands and options
- Improved error handling across all packages

### Fixed
- File watcher handling multiple events correctly
- Directory open operations properly return errors
- Path utilities handle edge cases (empty paths, hidden files)
- CLI flag conflicts resolved

### Refactored
- Extracted common SQLite store functionality into base class, reducing code duplication by ~470 lines
- Fixed Go naming conventions by removing type stuttering (e.g., `CloudProvider` → `Provider`)
- Centralized CLI error handling into reusable utility functions
- Improved code organization and maintainability

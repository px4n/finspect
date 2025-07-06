# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial VFS (Virtual File System) implementation
- Local filesystem adaptor with full POSIX operations
- CLI with basic commands (mount, ls, cp, mv, rm, stat)
- File watching capabilities using fsnotify
- Path manipulation utilities
- Comprehensive test suite
- GitHub Actions CI/CD pipeline
- Project documentation and contributing guidelines

### Changed
- Complete rewrite from the original Perkeep-based implementation
- Simplified architecture focusing on VFS abstraction
- Modular adaptor system for future extensibility

### Fixed
- File watcher handling multiple events correctly
- Directory open operations properly return errors
- Path utilities handle edge cases (empty paths, hidden files)
- CLI flag conflicts resolved

## [0.1.0] - TBD

Initial release (planned)
# Contributing to finspect

Thanks for your interest in contributing to finspect. I appreciate any help you can provide to make this project better.

## How to Contribute

### Reporting Bugs

If you find a bug, please create an issue on GitHub with the following information:

- A clear, descriptive title
- Steps to reproduce the issue
- Expected behavior
- Actual behavior
- Your environment (OS, Go version, etc.)
- Any relevant logs or error messages

### Suggesting Features

Feature suggestions are welcome. Please create an issue with:

- A clear description of the feature
- Use cases for the feature
- Any implementation ideas you might have

### Code Contributions

1. **Fork the Repository**

   ```bash
   git clone https://github.com/px4n/finspect.git
   cd finspect
   ```

2. **Create a Branch**

   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/your-bug-fix
   ```

3. **Make Your Changes**

   - Follow the existing code style
   - Add tests for new functionality
   - Update documentation as needed

4. **Test Your Changes**

   ```bash
   # Run all tests
   make test

   # Run the CLI tests
   ./test_cli.sh

   # Build and test manually
   make build
   ./bin/finspect version
   ```

5. **Commit Your Changes**

   ```bash
   git add .
   git commit -m "feat: add support for X"
   # or "fix: resolve issue with Y"
   # or "docs: update README"
   ```

   Follow conventional commit format:

   - `feat:` New feature
   - `fix:` Bug fix
   - `docs:` Documentation changes
   - `test:` Test additions or modifications
   - `refactor:` Code refactoring
   - `style:` Code style changes
   - `chore:` Build process or auxiliary tool changes

6. **Push and Create a Pull Request**
   ```bash
   git push origin feature/your-feature-name
   ```
   Then create a pull request on GitHub.

## Development Guidelines

### Code Style

- Follow standard Go conventions
- Use `gofmt` to format your code
- Use meaningful variable and function names
- Add comments for complex logic
- Keep functions small and focused

### Testing

- Write unit tests for new functionality
- Ensure all tests pass before submitting PR
- Aim for good test coverage
- Include integration tests where appropriate

### Documentation

- Update README.md if adding new features
- Add godoc comments to exported functions
- Update relevant documentation in the `docs/` folder
- Include examples in documentation

## Areas for Contribution

### High Priority

- **New Adaptors**: Cloud storage (Google Drive, Dropbox), social media (Facebook, Twitter)
- **Search Implementation**: Full-text search across mounted sources
- **Metadata System**: Flexible tagging and metadata storage

### Medium Priority

- **Web UI**: Browser-based interface for file management
- **Performance**: Optimization of file operations
- **Configuration**: Better config management and persistence

### Good First Issues

- Adding more CLI commands
- Improving error messages
- Writing more tests
- Documentation improvements

## Architecture Overview

Before contributing, familiarize yourself with the architecture:

```
pkg/vfs/          - Core VFS interfaces and router
adaptors/         - Data source implementations
cmd/finspect/     - CLI application
internal/         - Internal packages
docs/             - Documentation
```

## Pull Request Process

1. Ensure all tests pass
2. Update documentation
3. Add yourself to CONTRIBUTORS.md (if not already there)
4. Submit PR with clear description
5. Wait for review and address feedback

## Questions?

Feel free to:

- Open an issue for questions
- Start a discussion in GitHub Discussions
- Reach out to maintainers

Thanks for contributing to finspect.

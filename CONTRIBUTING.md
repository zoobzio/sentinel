# Contributing to Sentinel

Thank you for your interest in contributing to Sentinel! We welcome contributions from the community.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/your-username/sentinel.git`
3. Create a new branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Run tests: `make test`
6. Run linters: `make lint`
7. Commit your changes with a descriptive message
8. Push to your fork and submit a pull request

## Development Setup

### Prerequisites
- Go 1.24 or higher
- golangci-lint (install with `make install-tools`)

### Running Tests
```bash
make test        # Run all tests
make test-bench  # Run benchmarks
make coverage    # Generate coverage report
make lint        # Run linters
make check       # Run tests and lint
```

## Code Style

- Follow standard Go conventions
- Ensure all code passes `golangci-lint`
- Write tests for new functionality
- Keep test coverage above 90%
- Document exported functions and types

## Pull Request Process

1. Ensure all tests pass
2. Update documentation if needed
3. Add entries to CHANGELOG.md if applicable
4. Ensure your PR description clearly describes the problem and solution
5. Link any relevant issues

## Testing Guidelines

- Each source file should have a corresponding test file
- Write both positive and negative test cases
- Use table-driven tests where appropriate
- Ensure tests are deterministic and don't depend on external services

## Commit Message Format

Use conventional commits format:
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Test additions or changes
- `refactor:` Code refactoring
- `perf:` Performance improvements
- `chore:` Maintenance tasks

## Questions?

Feel free to open an issue for any questions or concerns.
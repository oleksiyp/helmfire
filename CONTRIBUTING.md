# Contributing to Helmfire

Thank you for your interest in contributing to Helmfire! This guide will help you get started.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Code Style](#code-style)
- [Documentation](#documentation)

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for all contributors.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/helmfire.git
   cd helmfire
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/oleksiyp/helmfire.git
   ```

## Development Setup

### Prerequisites

- Go 1.21 or higher
- `helm` (version 3.x)
- `kubectl` (for integration tests)
- Access to a Kubernetes cluster (for integration tests)
- `golangci-lint` (for linting)

### Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Build

```bash
make build
```

This creates the `helmfire` binary in the current directory.

## Project Structure

```
helmfire/
├── cmd/
│   └── helmfire/          # Main application entry point
├── pkg/
│   ├── drift/             # Drift detection implementation
│   ├── helmstate/         # Helmfile state management
│   ├── substitute/        # Chart/image substitution manager
│   └── sync/              # Release synchronization
├── internal/
│   └── version/           # Version information
├── test/                  # Integration and E2E tests
├── examples/              # Example configurations
└── docs/                  # Additional documentation
```

## Making Changes

### Create a Branch

Always create a new branch for your changes:

```bash
git checkout -b feature/your-feature-name
```

Use descriptive branch names:
- `feature/` for new features
- `fix/` for bug fixes
- `docs/` for documentation changes
- `test/` for test improvements

### Development Workflow

1. **Write code** following our [code style](#code-style)
2. **Add tests** for your changes (required for features and fixes)
3. **Run tests** to ensure everything works:
   ```bash
   make test
   ```
4. **Run linter**:
   ```bash
   make lint
   ```
5. **Update documentation** if needed

## Testing

### Unit Tests

Run unit tests for all packages:

```bash
go test -v -race -cover ./...
```

Run tests for a specific package:

```bash
go test -v ./pkg/substitute/
```

### Integration Tests

Integration tests require helm and kubectl:

```bash
go test -v ./test/
```

### End-to-End Tests

E2E tests require a Kubernetes cluster:

```bash
go test -v -tags=e2e ./test/
```

### Benchmarks

Run performance benchmarks:

```bash
go test -bench=. -benchmem ./test/
```

### Coverage

Generate coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Submitting Changes

### Commit Guidelines

Write clear, descriptive commit messages:

```
<type>: <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test improvements
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Build/tooling changes

**Example:**
```
feat: add support for helm diff plugin in drift detection

- Integrate helm-diff plugin for more accurate drift detection
- Add configuration option for custom diff command
- Update documentation with new drift detection options

Closes #123
```

### Pull Request Process

1. **Update your branch** with latest upstream:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Push your changes**:
   ```bash
   git push origin feature/your-feature-name
   ```

3. **Create a Pull Request** on GitHub with:
   - Clear title describing the change
   - Description of what changed and why
   - Link to related issues
   - Screenshots for UI changes (if applicable)

4. **Address review feedback**:
   - Make requested changes
   - Push updates to your branch
   - Respond to comments

5. **Ensure CI passes**:
   - All tests must pass
   - Linter must pass
   - Coverage should not decrease

## Code Style

### Go Standards

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting (automatic with most editors)
- Run `golangci-lint` before committing

### Best Practices

1. **Error Handling**
   ```go
   // Good
   if err := doSomething(); err != nil {
       return fmt.Errorf("failed to do something: %w", err)
   }

   // Bad
   err := doSomething()
   if err != nil {
       return err
   }
   ```

2. **Logging**
   ```go
   // Use structured logging with zap
   logger.Info("syncing release",
       zap.String("name", release.Name),
       zap.String("namespace", namespace))
   ```

3. **Testing**
   ```go
   // Use table-driven tests
   tests := []struct {
       name     string
       input    string
       expected string
   }{
       {"case 1", "input1", "expected1"},
       {"case 2", "input2", "expected2"},
   }

   for _, tt := range tests {
       t.Run(tt.name, func(t *testing.T) {
           result := function(tt.input)
           if result != tt.expected {
               t.Errorf("expected %v, got %v", tt.expected, result)
           }
       })
   }
   ```

4. **Naming**
   - Use descriptive names
   - Package names: lowercase, single word
   - Interfaces: noun or adjective (e.g., `Reader`, `Executable`)
   - Functions: verb or verb phrase (e.g., `LoadFile`, `SyncRelease`)

### Comments

- Add godoc comments for exported functions/types
- Explain "why" not "what" in comments
- Keep comments up to date with code changes

```go
// NewManager creates a new substitution manager with empty registries.
// Chart and image substitutions must be added separately using
// AddChartSubstitution and AddImageSubstitution methods.
func NewManager() *Manager {
    return &Manager{
        charts: make(map[string]string),
        images: make(map[string]string),
    }
}
```

## Documentation

### Update Documentation

When making changes, update relevant documentation:

- **README.md**: User-facing features, installation, quick start
- **HELMFIRE_ARCHITECTURE.md**: Architecture changes
- **Code comments**: Godoc for exported functions
- **Examples**: Add or update examples in `examples/`

### Documentation Standards

- Use clear, concise language
- Include code examples
- Add diagrams where helpful (use ASCII or mermaid)
- Keep documentation in sync with code

## Release Process

Releases are managed by maintainers. The process is:

1. Update version in `internal/version/version.go`
2. Update CHANGELOG.md
3. Create and push a version tag
4. GitHub Actions automatically builds and publishes release artifacts

## Getting Help

- **Questions**: Open a GitHub Discussion
- **Bugs**: Open a GitHub Issue with reproduction steps
- **Security**: Email security@helmfire.dev (do not open public issues)

## Recognition

Contributors are recognized in:
- GitHub contributors page
- Release notes
- Project README

Thank you for contributing to Helmfire!

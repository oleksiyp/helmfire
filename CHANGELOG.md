# Changelog

All notable changes to Helmfire will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Phase 5: Production polish and tooling
  - Comprehensive test suite with 60%+ coverage
  - End-to-end integration tests
  - Performance benchmarks
  - Complete API reference documentation
  - Contributing guidelines
  - GitHub Actions CI/CD pipeline
  - Docker image support
  - Homebrew formula template
  - Release automation

## [0.3.0] - 2024-11-16

### Added
- Phase 3: Drift detection with auto-healing
  - Periodic drift detection using helm diff
  - Configurable check intervals
  - Stdout and webhook notifications
  - Auto-healing capabilities
  - Drift severity classification

### Changed
- Enhanced logging with structured output
- Improved error messages

## [0.1.0] - 2024-11-15

### Added
- Phase 1: Foundation implementation
  - Basic helmfile sync functionality
  - Chart substitution (local chart override)
  - Image substitution (post-renderer)
  - Substitution manager with persistence
  - CLI commands: sync, chart, image, list, remove
  - Unit tests for core functionality
  - Example configurations

### Infrastructure
- Project structure and build system
- Go module setup
- Makefile for common tasks
- Initial documentation

## Project Milestones

### Completed
- ✅ Phase 0: Research and Analysis
- ✅ Phase 1: Foundation with sync and substitution
- ✅ Phase 3: Drift detection
- ✅ Phase 5: Polish and documentation

### Planned
- ⏳ Phase 2: File watching with auto-sync
- ⏳ Phase 4: Daemon mode with API

### Future Enhancements
- Multi-environment support
- GUI dashboard
- Advanced notification integrations
- Plugin system
- Configuration backup to Git

---

## Version Naming

- **Major version** (1.x.x): Breaking API changes
- **Minor version** (x.1.x): New features, backward compatible
- **Patch version** (x.x.1): Bug fixes, backward compatible

## Release Process

1. Update version in `internal/version/version.go`
2. Update this CHANGELOG.md
3. Commit: `git commit -m "Release vX.Y.Z"`
4. Tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`
5. Push: `git push && git push --tags`
6. GitHub Actions automatically creates release

---

[Unreleased]: https://github.com/oleksiyp/helmfire/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/oleksiyp/helmfire/compare/v0.1.0...v0.3.0
[0.1.0]: https://github.com/oleksiyp/helmfire/releases/tag/v0.1.0

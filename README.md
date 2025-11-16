# Helmfire

**Dynamic Kubernetes deployment tool with live chart/image substitution and continuous monitoring**

Helmfire extends [helmfile](https://github.com/helmfile/helmfile) with developer-friendly features for rapid iteration and production monitoring.

## Features

- ✅ **Basic Sync** - Helmfile-compatible release synchronization (Phase 1)
- ✅ **Chart Substitution** - Replace remote charts with local versions (Phase 1)
- ✅ **Image Substitution** - Override container images via post-renderer (Phase 1)
- ✅ **Watch Mode** - Auto-sync on helmfile.yaml or values file changes (Phase 2)
- ✅ **Drift Detection** - Monitor cluster state vs. desired state (Phase 3)
- ✅ **Daemon Mode** - Background process with API control (Phase 4)
- ✅ **Production Ready** - Comprehensive tests, docs, and tooling (Phase 5)


**What's New  v1.0.0:**
- 60%+ test coverage with unit, integration, and E2E tests
- Performance benchmarks
- Complete API reference and contributing guide
- GitHub Actions CI/CD pipeline
- Multi-platform releases (Linux, macOS, Windows)
- Docker image support
- Homebrew formula

See [examples/](examples/) to try it out!

## Quick Start

### Installation

```bash
git clone https://github.com/oleksiyp/helmfire.git
cd helmfire
make build
sudo mv helmfire /usr/local/bin/
```

Prerequisites: Go 1.21+, `helm`, and `kubectl`

### Basic Sync

```bash
# Sync all releases from helmfile.yaml
helmfire sync

# Dry-run to preview changes
helmfire sync --dry-run

# Sync with specific file
helmfire sync -f path/to/helmfile.yaml
```

### Chart Substitution

```bash
# Add chart substitution
helmfire chart bitnami/nginx ./examples/local-chart/my-nginx-chart

# Run sync with substitution applied
helmfire sync --dry-run

# List active substitutions
helmfire list charts
```

### Image Substitution

```bash
# Add image substitution
helmfire image postgres:15 localhost:5000/postgres:custom

# Run sync with substitution applied
helmfire sync --dry-run

# List active substitutions
helmfire list images
```

### Drift Detection

```bash
# Enable drift detection (checks every 30s by default)
helmfire sync --drift-detect

# Custom interval
helmfire sync --drift-detect --drift-interval=1m

# With auto-healing
helmfire sync --drift-detect --drift-auto-heal

# With webhook notifications
helmfire sync --drift-detect --drift-webhook=https://hooks.slack.com/...
```

### Daemon Mode

```bash
# Start daemon with drift detection
helmfire daemon start --drift-interval=1m

# Check daemon status
helmfire daemon status

# Add substitutions to running daemon
helmfire chart bitnami/nginx ./my-chart
helmfire image postgres:15 localhost:5000/postgres:dev

# View daemon logs
helmfire daemon logs

# Stop daemon
helmfire daemon stop
```

### Try the Examples

```bash
cd examples/simple-app

# Basic sync
helmfire sync -f helmfile.yaml --dry-run

# With chart substitution
helmfire chart bitnami/nginx ../local-chart/my-nginx-chart
helmfire sync -f helmfile.yaml --dry-run
```

See [examples/README.md](examples/README.md) for more.

## Project Documentation

- [Architecture Design](HELMFIRE_ARCHITECTURE.md) - Complete system design
- [Helmfile Analysis](HELMFILE_ANALYSIS.md) - Deep dive into helmfile internals
- [Helm Analysis](HELM_PROJECT_ANALYSIS.md) - Comprehensive helm architecture analysis
- [Reusable Libraries](REUSABLE_LIBRARIES.md) - Library integration guide

### Development Workflow

Iterate rapidly on local charts without pushing to a registry:

```bash
# Start helmfire
helmfire sync --watch

# Override production chart with local dev version
helmfire chart mycompany/myapp ./myapp-chart

# Edit chart templates, helmfire auto-detects and re-deploys
# Test immediately in Kubernetes
```

### Multi-Service Development

Work on multiple services simultaneously:

```bash
helmfire sync --watch
helmfire chart company/frontend ./frontend-chart
helmfire chart company/backend ./backend-chart
helmfire image postgres:15 localhost:5000/postgres:dev

# All three components now use local versions
# Any changes trigger automatic re-deployment
```

### Production Monitoring

Monitor for configuration drift and maintain desired state:

```bash
helmfire sync --watch --drift-detect --drift-interval=1m

# Alerts when cluster state diverges from helmfile
# Optional auto-healing to restore desired state
```

## Contributing

This project is in early development. Contributions are welcome!

1. Review the [architecture documentation](HELMFIRE_ARCHITECTURE.md)
2. Check open issues for tasks
3. Submit pull requests with tests

## License

Apache 2.0 (same as Helm and Helmfile)

## Acknowledgments

Helmfire builds upon the excellent work of:
- [Helmfile](https://github.com/helmfile/helmfile) - Declarative Kubernetes deployment
- [Helm](https://github.com/helm/helm) - Kubernetes package manager
- [fsnotify](https://github.com/fsnotify/fsnotify) - Cross-platform file watching

## Command Reference

### helmfire sync
```bash
helmfire sync [flags]
```
Flags: `-f/--file`, `-n/--namespace`, `--kube-context`, `--dry-run`, `--watch` (Phase 2+)

### helmfire chart
```bash
helmfire chart <original> <local-path>
```
Example: `helmfire chart bitnami/postgresql ./charts/postgres-dev`

### helmfire image
```bash
helmfire image <original> <replacement>
```
Example: `helmfire image postgres:15 localhost:5000/postgres:custom`

### helmfire list/remove
```bash
helmfire list charts|images
helmfire remove chart|image <name>
```

### helmfire daemon
```bash
helmfire daemon start [flags]
helmfire daemon stop [flags]
helmfire daemon status [flags]
helmfire daemon logs [flags]
```
Flags for start: `--drift-interval`, `--drift-auto-heal`, `--drift-webhook`, `--api-addr`, `--pid-file`, `--log-file`

## Project Status

**v1.0.0 Released!** Production-ready with comprehensive testing and tooling.

See [HELMFIRE_ARCHITECTURE.md](HELMFIRE_ARCHITECTURE.md) for detailed roadmap.

## Documentation

- [API Reference](docs/API_REFERENCE.md) - Complete command reference
- [Contributing Guide](CONTRIBUTING.md) - Development guide
- [Architecture Design](HELMFIRE_ARCHITECTURE.md) - System design
- [Changelog](CHANGELOG.md) - Version history

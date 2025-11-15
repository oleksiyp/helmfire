# Helmfire

**Dynamic Kubernetes deployment tool with live chart/image substitution and continuous monitoring**

Helmfire extends [helmfile](https://github.com/helmfile/helmfile) with developer-friendly features for rapid iteration and production monitoring.

## Features

- 🔄 **Watch Mode** - Auto-sync on helmfile.yaml or values file changes
- 🔧 **Live Chart Substitution** - Replace remote charts with local versions on-the-fly
- 🎯 **Live Image Substitution** - Override container images dynamically
- 📊 **Drift Detection** - Monitor cluster state vs. desired state
- 🤖 **Daemon Mode** - Background process with API control
- ⚡ **Selective Sync** - Only re-deploy affected releases on changes

## Status

🚧 **Under Development** 🚧

This project is currently in the design and initial implementation phase. See the [architecture documentation](HELMFIRE_ARCHITECTURE.md) for details.

## Quick Start

> Note: These examples show the intended usage. Implementation is in progress.

### Basic Sync with Watching

```bash
# Start helmfire in watch mode
helmfire sync --watch

# In another terminal, edit your helmfile or charts
# Helmfire automatically detects changes and re-syncs
```

### Chart Substitution

```bash
# Replace a remote chart with local version for development
helmfire chart bitnami/postgresql ./charts/postgresql-dev

# All releases using bitnami/postgresql will now use your local version
# Changes to ./charts/postgresql-dev trigger automatic re-deployment
```

### Image Substitution

```bash
# Replace an image across all releases
helmfire image postgres:15 localhost:5000/postgres:my-custom-build

# Test your custom image without modifying helmfile or values
```

### Drift Detection

```bash
# Monitor for configuration drift
helmfire sync --watch --drift-detect --drift-interval=30s

# Auto-heal drift (restore desired state)
helmfire sync --watch --drift-detect --drift-auto-heal
```

## Project Documentation

- [Architecture Design](HELMFIRE_ARCHITECTURE.md) - Complete system design
- [Helmfile Analysis](HELMFILE_ANALYSIS.md) - Deep dive into helmfile internals
- [Helm Analysis](HELM_PROJECT_ANALYSIS.md) - Comprehensive helm architecture analysis
- [Reusable Libraries](REUSABLE_LIBRARIES.md) - Library integration guide

## Development Phases

- [x] Phase 0: Research and Analysis
  - [x] Analyze helmfile source code
  - [x] Analyze helm source code
  - [x] Design architecture
  - [x] Identify reusable components
- [ ] Phase 1: Foundation (Weeks 1-2)
  - [ ] Project setup and structure
  - [ ] Substitution Manager implementation
  - [ ] Basic sync command
  - [ ] Chart/image substitution commands
- [ ] Phase 2: File Watching (Weeks 3-4)
  - [ ] File watcher implementation
  - [ ] Debouncing logic
  - [ ] Selective sync
- [ ] Phase 3: Drift Detection (Week 5)
  - [ ] Drift detector implementation
  - [ ] Notification system
  - [ ] Auto-healing
- [ ] Phase 4: Daemon Mode (Week 6)
  - [ ] Background process
  - [ ] API server
  - [ ] Control commands
- [ ] Phase 5: Polish (Weeks 7-8)
  - [ ] Testing
  - [ ] Documentation
  - [ ] Release automation

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                    Helmfire CLI                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐             │
│  │   sync   │  │  chart   │  │  image   │             │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘             │
└───────┼─────────────┼─────────────┼────────────────────┘
        │             │             │
        └─────────────┴─────────────┘
                      │
┌─────────────────────┼─────────────────────────────────┐
│         Helmfire Core Engine                          │
│  ┌──────────────────────────────────────────┐         │
│  │   Substitution Manager                   │         │
│  │   - Charts: remote → local mappings      │         │
│  │   - Images: original → replacement       │         │
│  └──────────────────────────────────────────┘         │
│  ┌──────────────────────────────────────────┐         │
│  │   File Watcher (fsnotify)                │         │
│  │   - Debouncing                           │         │
│  │   - Change detection                     │         │
│  └──────────────────────────────────────────┘         │
│  ┌──────────────────────────────────────────┐         │
│  │   Helmfile State Manager                 │         │
│  │   - Parse helmfile.yaml                  │         │
│  │   - DAG planning                         │         │
│  └──────────────────────────────────────────┘         │
│  ┌──────────────────────────────────────────┐         │
│  │   Drift Detector                         │         │
│  │   - Periodic diff                        │         │
│  │   - Auto-healing                         │         │
│  └──────────────────────────────────────────┘         │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────┼────────────────────────────────────┐
│         External Dependencies                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │ Helmfile │  │   Helm   │  │    K8s   │            │
│  └──────────┘  └──────────┘  └──────────┘            │
└─────────────────────────────────────────────────────────┘
```

## Comparison with Helmfile

| Feature | Helmfile | Helmfire |
|---------|----------|----------|
| Declarative releases | ✅ | ✅ |
| DAG-based deployment | ✅ | ✅ |
| Values management | ✅ | ✅ |
| Lifecycle hooks | ✅ | ✅ |
| File watching | ❌ | ✅ |
| Auto-reload | ❌ | ✅ |
| Chart substitution | ❌ | ✅ |
| Image substitution | ❌ | ✅ |
| Drift detection | ❌ | ✅ |
| Daemon mode | ❌ | ✅ |

## Use Cases

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

## Project Status

Current focus: Phase 1 - Foundation

See [HELMFIRE_ARCHITECTURE.md](HELMFIRE_ARCHITECTURE.md) for detailed implementation plan.

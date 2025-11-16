# Helmfire API Reference

Complete reference for Helmfire commands, flags, and configuration.

## Table of Contents

- [Commands](#commands)
  - [helmfire sync](#helmfire-sync)
  - [helmfire chart](#helmfire-chart)
  - [helmfire image](#helmfire-image)
  - [helmfire list](#helmfire-list)
  - [helmfire remove](#helmfire-remove)
  - [helmfire version](#helmfire-version)
- [Flags](#flags)
- [Configuration](#configuration)
- [Exit Codes](#exit-codes)

## Commands

### helmfire sync

Synchronize releases defined in helmfile.yaml to the Kubernetes cluster.

**Synopsis:**
```bash
helmfire sync [flags]
```

**Description:**

Executes helmfile sync with optional watching, drift detection, and auto-healing capabilities.

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-f, --file` | string | `helmfile.yaml` | Path to helmfile |
| `-e, --environment` | string | `` | Environment name |
| `-l, --selector` | string | `` | Label selector (e.g., `app=web`) |
| `-n, --namespace` | string | `` | Default namespace |
| `--kube-context` | string | `` | Kubernetes context to use |
| `--dry-run` | bool | `false` | Simulate sync without applying changes |
| `--watch` | bool | `false` | Watch for changes and auto-sync |
| `--drift-detect` | bool | `false` | Enable drift detection |
| `--drift-interval` | duration | `30s` | Drift check interval |
| `--drift-auto-heal` | bool | `false` | Automatically heal detected drift |
| `--drift-webhook` | string | `` | Webhook URL for drift notifications |

**Examples:**

```bash
# Basic sync
helmfire sync

# Sync with specific file
helmfire sync -f path/to/helmfile.yaml

# Dry-run to preview changes
helmfire sync --dry-run

# Sync with label selector
helmfire sync -l tier=frontend

# Watch mode with drift detection
helmfire sync --watch --drift-detect --drift-interval=1m

# Auto-healing mode
helmfire sync --drift-detect --drift-auto-heal
```

**Exit Codes:**
- `0`: Success
- `1`: Sync failed
- `2`: File not found
- `3`: Invalid configuration

---

### helmfire chart

Add or update chart substitution mapping.

**Synopsis:**
```bash
helmfire chart <original-chart> <local-path>
```

**Description:**

Maps a remote chart reference to a local chart directory. When syncing, helmfire will use the local chart instead of the remote one.

**Arguments:**

| Argument | Description |
|----------|-------------|
| `original-chart` | Original chart reference (e.g., `bitnami/nginx`) |
| `local-path` | Path to local chart directory |

**Examples:**

```bash
# Substitute bitnami/nginx with local chart
helmfire chart bitnami/nginx ./charts/my-nginx

# Use absolute path
helmfire chart stable/postgresql /home/user/charts/postgres-dev

# Works with any chart reference
helmfire chart myrepo/myapp ../myapp-chart
```

**Validation:**

- Local path must exist
- Local path must contain a valid Chart.yaml
- Chart name in Chart.yaml doesn't need to match original

**Notes:**

- Substitutions are stored in `~/.helmfire/substitutions.yaml`
- Multiple substitutions can be active simultaneously
- Changes to local chart trigger auto-sync if `--watch` is enabled

---

### helmfire image

Add or update image substitution mapping.

**Synopsis:**
```bash
helmfire image <original-image> <replacement-image>
```

**Description:**

Maps a container image reference to a replacement. When syncing, helmfire will replace all occurrences of the original image with the replacement using a post-renderer.

**Arguments:**

| Argument | Description |
|----------|-------------|
| `original-image` | Original image reference (e.g., `nginx:1.21`) |
| `replacement-image` | Replacement image reference |

**Examples:**

```bash
# Replace nginx version
helmfire image nginx:1.21 nginx:1.22

# Use local registry
helmfire image postgres:15 localhost:5000/postgres:custom

# Use different repository
helmfire image bitnami/nginx:latest myregistry.io/nginx:stable
```

**Image Reference Formats:**

Supported formats:
- `name:tag` (e.g., `nginx:1.21`)
- `repository/name:tag` (e.g., `bitnami/nginx:latest`)
- `registry/repository/name:tag` (e.g., `gcr.io/project/app:v1`)

**Notes:**

- Substitutions are applied to all container types (Deployment, StatefulSet, DaemonSet, Job, Pod)
- Affects both `containers` and `initContainers`
- Does not modify image pull policy

---

### helmfire list

List active substitutions.

**Synopsis:**
```bash
helmfire list <charts|images>
```

**Description:**

Display all active chart or image substitutions.

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `charts` | List chart substitutions |
| `images` | List image substitutions |

**Examples:**

```bash
# List chart substitutions
helmfire list charts

# List image substitutions
helmfire list images
```

**Output Format:**

```
Chart Substitutions:
  bitnami/nginx -> /home/user/charts/my-nginx
  stable/postgresql -> /home/user/charts/postgres-dev

Image Substitutions:
  nginx:1.21 -> nginx:1.22
  postgres:15 -> localhost:5000/postgres:custom
```

---

### helmfire remove

Remove a substitution.

**Synopsis:**
```bash
helmfire remove <chart|image> <name>
```

**Description:**

Remove an active chart or image substitution.

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `chart` | Remove chart substitution |
| `image` | Remove image substitution |

**Examples:**

```bash
# Remove chart substitution
helmfire remove chart bitnami/nginx

# Remove image substitution
helmfire remove image nginx:1.21
```

---

### helmfire version

Display version information.

**Synopsis:**
```bash
helmfire version
```

**Description:**

Show helmfire version, git commit, and build date.

**Example:**

```bash
helmfire version
```

**Output:**
```
Helmfire version: 1.0.0
Git commit: a1b2c3d
Build date: 2024-01-15T10:30:00Z
```

---

## Flags

### Global Flags

Available for all commands:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--log-level` | string | `info` | Log level (debug, info, warn, error) |
| `--no-color` | bool | `false` | Disable colored output |
| `-h, --help` | bool | `false` | Show help |

---

## Configuration

### Configuration File

Helmfire can be configured via `~/.helmfire/config.yaml`:

```yaml
# Default helmfile path
helmfilePath: helmfile.yaml

# Default environment
environment: development

# Drift detection settings
drift:
  enabled: false
  interval: 30s
  autoHeal: false
  webhook: ""

# Watch mode settings
watch:
  enabled: false
  debounce: 500ms

# Logging
logging:
  level: info
  format: text  # or json

# Substitutions (managed automatically)
substitutions:
  charts:
    bitnami/nginx: /path/to/local/chart
  images:
    nginx:1.21: nginx:1.22
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `HELMFIRE_CONFIG` | Config file path | `~/.helmfire/config.yaml` |
| `HELMFIRE_LOG_LEVEL` | Log level | `info` |
| `HELMFILE_PATH` | Default helmfile path | `helmfile.yaml` |
| `KUBECONFIG` | Kubernetes config | `~/.kube/config` |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | File not found |
| 3 | Invalid configuration |
| 4 | Helm command failed |
| 5 | Kubernetes API error |
| 10 | Drift detected (with `--drift-detect` and no auto-heal) |

---

## Examples

### Development Workflow

Start helmfire with watch mode and use local chart:

```bash
# Terminal 1: Start helmfire
helmfire sync --watch

# Terminal 2: Add substitution
helmfire chart bitnami/nginx ./my-nginx-chart

# Edit ./my-nginx-chart/templates/*
# Helmfire auto-detects changes and re-syncs
```

### Production Monitoring

Monitor for drift and auto-heal:

```bash
helmfire sync \
  --drift-detect \
  --drift-interval=1m \
  --drift-auto-heal \
  --drift-webhook=https://hooks.slack.com/services/YOUR/WEBHOOK/URL
```

### Multi-Service Development

Override multiple charts and images:

```bash
# Start watching
helmfire sync --watch

# Override frontend chart
helmfire chart company/frontend ./frontend-chart

# Override backend chart
helmfire chart company/backend ./backend-chart

# Override database image
helmfire image postgres:15 localhost:5000/postgres:dev

# All changes auto-sync
```

---

## See Also

- [README.md](../README.md) - Quick start and overview
- [HELMFIRE_ARCHITECTURE.md](../HELMFIRE_ARCHITECTURE.md) - Architecture details
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Development guide

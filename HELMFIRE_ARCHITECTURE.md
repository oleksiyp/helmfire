# Helmfire Architecture Design

**Project Vision:** A dynamic Kubernetes deployment tool that extends helmfile with file watching, live chart/image substitution, and continuous drift detection.

**Based on Analysis of:**
- Helmfile (see HELMFILE_ANALYSIS.md)
- Helm (see HELM_PROJECT_ANALYSIS.md)

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Core Features](#core-features)
3. [Architecture Overview](#architecture-overview)
4. [Component Design](#component-design)
5. [Reusable Libraries](#reusable-libraries)
6. [Data Flow](#data-flow)
7. [API Design](#api-design)
8. [Implementation Phases](#implementation-phases)
9. [Technical Decisions](#technical-decisions)

---

## Executive Summary

### What is Helmfire?

Helmfire is a development and operations tool that:
1. **Runs helmfile sync** with continuous monitoring
2. **Watches for changes** in helmfile.yaml, values files, and local charts
3. **Detects drift** between desired state and cluster state
4. **Substitutes resources dynamically** while running:
   - `helmfire chart <repo>/<name> <local-path>` - Swap remote chart with local version
   - `helmfire image <original> <replacement>` - Override container images

### Key Differentiators

| Feature | Helmfile | Helmfire |
|---------|----------|----------|
| Sync releases | ✅ | ✅ |
| File watching | ❌ | ✅ |
| Auto-reload on changes | ❌ | ✅ |
| Live chart substitution | ❌ | ✅ |
| Live image substitution | ❌ | ✅ |
| Drift detection | ❌ | ✅ |
| Background daemon mode | ❌ | ✅ |

### Use Cases

1. **Development Workflow:**
   ```bash
   # Terminal 1: Start helmfire with local chart override
   helmfire sync --watch
   helmfire chart bitnami/postgresql ./charts/postgresql-dev

   # Edit local chart, helmfire auto-detects and re-deploys
   # Edit helmfile.yaml, helmfire auto-syncs
   ```

2. **Image Testing:**
   ```bash
   helmfire sync --watch
   helmfire image postgres:15 localhost:5000/postgres:my-test
   # All postgres:15 references replaced with local build
   ```

3. **Continuous Operations:**
   ```bash
   helmfire sync --watch --drift-detect
   # Monitors for configuration drift, auto-heals if configured
   ```

---

## Core Features

### 1. Helmfile Sync (Foundation)

**Status:** Reuse helmfile libraries

**Capabilities:**
- Parse helmfile.yaml with all helmfile features
- Execute DAG-based release deployment
- Support all helmfile options (selectors, environments, values)
- Helm repository management
- Chart preparation and dependency resolution

**Implementation:**
- Import `github.com/helmfile/helmfile/pkg/state`
- Import `github.com/helmfile/helmfile/pkg/app`
- Import `github.com/helmfile/helmfile/pkg/helmexec`

### 2. File Watching

**Status:** New implementation

**Watched Resources:**
- `helmfile.yaml` and subhelmfiles
- Values files (`.yaml`, `.yml`)
- Local chart directories (when substituted)
- Chart.yaml and templates in local charts

**Technology:**
- `github.com/fsnotify/fsnotify` - Cross-platform file system notifications
- Debouncing to avoid rapid re-triggers
- Smart dependency tracking

**Behavior:**
```
File Change Detected
    ↓
Debounce (500ms)
    ↓
Parse Changed File
    ↓
Determine Impact (which releases affected)
    ↓
Selective Sync (only affected releases)
    ↓
Report Results
```

### 3. Dynamic Chart Substitution

**Status:** New implementation

**Command:** `helmfire chart <original> <local-path>`

**Example:**
```bash
# Substitute bitnami/postgresql with local chart
helmfire chart bitnami/postgresql ./charts/postgresql

# Multiple substitutions
helmfire chart stable/mysql ./charts/mysql
helmfire chart myrepo/myapp ./myapp-chart
```

**Implementation:**
- Intercept chart resolution in helmfile state
- Override chart path before template rendering
- Watch local chart directory for changes
- Trigger re-sync when local chart changes

**Technical Approach:**
- Modify `ReleaseSpec.Chart` field during state preparation
- Hook into `HelmState.PrepareCharts()` pipeline
- Use Helm's chart loader for local directories

### 4. Dynamic Image Substitution

**Status:** New implementation

**Command:** `helmfire image <original> <replacement>`

**Example:**
```bash
# Replace postgres image globally
helmfire image postgres:15 myregistry.io/postgres:custom

# Multiple replacements
helmfire image nginx:1.21 localhost:5000/nginx:dev
helmfire image redis:7 redis:7-alpine
```

**Implementation:**
- Post-render pipeline (after template rendering, before K8s apply)
- Parse rendered YAML manifests
- Find and replace image references in:
  - Deployment spec.template.spec.containers[].image
  - StatefulSet spec.template.spec.containers[].image
  - DaemonSet spec.template.spec.containers[].image
  - Job/CronJob spec.template.spec.containers[].image
  - Pod spec.containers[].image
  - Init containers and ephemeral containers

**Technical Approach:**
- Implement custom post-renderer using Helm's `pkg/postrenderer` interface
- Use `sigs.k8s.io/yaml` for safe YAML manipulation
- Maintain image substitution registry in memory

### 5. Drift Detection

**Status:** New implementation

**Capabilities:**
- Periodic comparison of desired vs. actual state
- Detect manual changes to deployed resources
- Alert on drift
- Optional auto-healing

**Implementation:**
- Use `HelmState.DiffReleases()` from helmfile
- Poll interval (configurable, default 30s)
- Calculate drift score
- Notification options: stdout, webhook, file

**Drift Types:**
```yaml
- Configuration Drift: Values changed in cluster
- Resource Drift: Replicas modified manually
- Image Drift: Container images changed
- Deletion Drift: Resources deleted manually
```

### 6. Daemon Mode

**Status:** New implementation

**Command:** `helmfire sync --watch --daemon`

**Features:**
- Background process with PID file
- Unix socket or HTTP API for control
- Log streaming
- Graceful shutdown
- Health checks

---

## Architecture Overview

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                        Helmfire CLI                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ sync command │  │ chart command│  │ image command│      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                  │                  │              │
│         └──────────────────┴──────────────────┘              │
│                            │                                 │
└────────────────────────────┼─────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                    Helmfire Core Engine                      │
│  ┌────────────────────────────────────────────────────┐     │
│  │              Substitution Manager                   │     │
│  │  - Chart Registry (original → local mappings)      │     │
│  │  - Image Registry (original → replacement)         │     │
│  └────────────────────────────────────────────────────┘     │
│                                                              │
│  ┌────────────────────────────────────────────────────┐     │
│  │               File Watcher                          │     │
│  │  - fsnotify integration                            │     │
│  │  - Debouncing logic                                │     │
│  │  - Change event processor                          │     │
│  └────────────────────────────────────────────────────┘     │
│                                                              │
│  ┌────────────────────────────────────────────────────┐     │
│  │            Helmfile State Manager                   │     │
│  │  - State loading (import helmfile/pkg/state)       │     │
│  │  - Release planning (DAG)                          │     │
│  │  - Chart preparation                               │     │
│  └────────────────────────────────────────────────────┘     │
│                                                              │
│  ┌────────────────────────────────────────────────────┐     │
│  │              Sync Orchestrator                      │     │
│  │  - Release execution                               │     │
│  │  - Hook management                                 │     │
│  │  - Concurrency control                             │     │
│  └────────────────────────────────────────────────────┘     │
│                                                              │
│  ┌────────────────────────────────────────────────────┐     │
│  │            Drift Detector                           │     │
│  │  - Periodic diff execution                         │     │
│  │  - Drift scoring                                   │     │
│  │  - Notification dispatch                           │     │
│  └────────────────────────────────────────────────────┘     │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                 External Dependencies                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Helmfile   │  │     Helm     │  │  Kubernetes  │      │
│  │  Libraries   │  │  Libraries   │  │    Cluster   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### Process Flow

```
                    ┌─────────────────┐
                    │  helmfire sync  │
                    │     --watch     │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │  Load helmfile  │
                    │  Parse releases │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ Apply chart/img │
                    │  substitutions  │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │  Plan releases  │
                    │   (build DAG)   │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │  Execute sync   │
                    │  (helm upgrade) │
                    └────────┬────────┘
                             │
                ┌────────────┴────────────┐
                │                         │
                ▼                         ▼
      ┌─────────────────┐      ┌─────────────────┐
      │  Start File     │      │ Start Drift     │
      │   Watcher       │      │   Detector      │
      └────────┬────────┘      └────────┬────────┘
               │                         │
               │                         │
               ▼                         ▼
      ┌─────────────────┐      ┌─────────────────┐
      │ Wait for change │      │   Wait 30s      │
      └────────┬────────┘      └────────┬────────┘
               │                         │
               │ Change detected         │ Timer triggered
               ▼                         ▼
      ┌─────────────────┐      ┌─────────────────┐
      │  Selective Re-  │      │   Run diff      │
      │      Sync       │      │  Check drift    │
      └────────┬────────┘      └────────┬────────┘
               │                         │
               └────────────┬────────────┘
                            │
                            ▼
                     ┌─────────────┐
                     │   Report    │
                     │   Results   │
                     └─────────────┘
```

---

## Component Design

### 1. Substitution Manager

**Package:** `pkg/substitute`

**Responsibilities:**
- Maintain chart substitution registry
- Maintain image substitution registry
- Apply substitutions to HelmState
- Apply substitutions to rendered manifests

**Data Structures:**

```go
package substitute

// Manager handles all resource substitutions
type Manager struct {
    charts map[string]string      // original chart → local path
    images map[string]string      // original image → replacement
    mu     sync.RWMutex
}

// ChartSubstitution represents a chart override
type ChartSubstitution struct {
    Original  string  // e.g., "bitnami/postgresql"
    LocalPath string  // e.g., "./charts/postgresql"
}

// ImageSubstitution represents an image override
type ImageSubstitution struct {
    Original    string  // e.g., "postgres:15"
    Replacement string  // e.g., "myregistry.io/postgres:custom"
}

func NewManager() *Manager
func (m *Manager) AddChartSubstitution(original, localPath string) error
func (m *Manager) AddImageSubstitution(original, replacement string) error
func (m *Manager) RemoveChartSubstitution(original string) error
func (m *Manager) RemoveImageSubstitution(original string) error
func (m *Manager) GetChartPath(original string) (string, bool)
func (m *Manager) GetImageReplacement(original string) (string, bool)
func (m *Manager) ApplyToState(state *state.HelmState) error
func (m *Manager) ApplyToManifests(manifests map[string]string) (map[string]string, error)
```

**Implementation Notes:**
- Thread-safe with mutex
- Validation of local paths (must exist)
- Validation of image references (must be valid)
- Chart substitution applied before sync
- Image substitution applied as post-render step

### 2. File Watcher

**Package:** `pkg/watcher`

**Responsibilities:**
- Monitor file system for changes
- Debounce rapid changes
- Determine affected releases
- Trigger selective sync

**Data Structures:**

```go
package watcher

import (
    "github.com/fsnotify/fsnotify"
    "github.com/helmfile/helmfile/pkg/state"
)

// Watcher monitors files for changes
type Watcher struct {
    fsWatcher    *fsnotify.Watcher
    debouncer    *Debouncer
    watchedPaths map[string]WatchedResource
    onChange     ChangeHandler
    ctx          context.Context
    cancel       context.CancelFunc
}

// WatchedResource tracks a monitored file/directory
type WatchedResource struct {
    Path         string
    Type         ResourceType  // helmfile, values, chart
    AffectedReleases []string   // which releases use this
}

type ResourceType string

const (
    ResourceTypeHelmfile ResourceType = "helmfile"
    ResourceTypeValues   ResourceType = "values"
    ResourceTypeChart    ResourceType = "chart"
)

// ChangeEvent represents a file system change
type ChangeEvent struct {
    Path             string
    Type             ResourceType
    AffectedReleases []string
}

type ChangeHandler func(event ChangeEvent) error

func NewWatcher(ctx context.Context, handler ChangeHandler) (*Watcher, error)
func (w *Watcher) AddHelmfile(path string) error
func (w *Watcher) AddValuesFile(path string, releases []string) error
func (w *Watcher) AddChartDirectory(path string, release string) error
func (w *Watcher) Start() error
func (w *Watcher) Stop() error
```

**Debouncing Strategy:**
- Collect all events within 500ms window
- Deduplicate events for same file
- Trigger single sync for all affected releases
- Configurable debounce duration

### 3. Helmfile State Manager

**Package:** `pkg/helmstate`

**Responsibilities:**
- Wrap helmfile state loading
- Apply substitutions
- Manage state lifecycle
- Provide state inspection

**Data Structures:**

```go
package helmstate

import (
    "github.com/helmfile/helmfile/pkg/state"
    "github.com/helmfile/helmfile/pkg/app"
    "github.com/helmfire/helmfire/pkg/substitute"
)

// Manager wraps helmfile state management
type Manager struct {
    app           *app.App
    substitutor   *substitute.Manager
    currentState  *state.HelmState
    helmfilePath  string
    environment   string
    logger        Logger
}

// LoadOptions configures state loading
type LoadOptions struct {
    FilePath    string
    Environment string
    Selectors   []string
    ValuesFiles []string
    SetValues   map[string]interface{}
}

func NewManager(opts LoadOptions, substitutor *substitute.Manager) (*Manager, error)
func (m *Manager) Load() (*state.HelmState, error)
func (m *Manager) Reload() (*state.HelmState, error)
func (m *Manager) GetState() *state.HelmState
func (m *Manager) GetAffectedReleases(changedFile string) []string
```

### 4. Sync Orchestrator

**Package:** `pkg/sync`

**Responsibilities:**
- Execute release synchronization
- Manage concurrency
- Handle hooks
- Report progress

**Data Structures:**

```go
package sync

import (
    "github.com/helmfile/helmfile/pkg/state"
    "github.com/helmfile/helmfile/pkg/app"
)

// Orchestrator manages sync operations
type Orchestrator struct {
    helmState    *state.HelmState
    app          *app.App
    concurrency  int
    dryRun       bool
    logger       Logger
}

// SyncOptions configures sync behavior
type SyncOptions struct {
    Concurrency    int
    DryRun         bool
    SkipDeps       bool
    Wait           bool
    Timeout        time.Duration
    Interactive    bool
    IncludeNeeds   bool
    Selectors      []string
}

// SyncResult captures sync outcome
type SyncResult struct {
    ReleaseName    string
    Namespace      string
    Success        bool
    Error          error
    Duration       time.Duration
    ChangesApplied bool
}

func NewOrchestrator(helmState *state.HelmState, app *app.App) *Orchestrator
func (o *Orchestrator) Sync(opts SyncOptions) ([]SyncResult, error)
func (o *Orchestrator) SyncReleases(releases []string, opts SyncOptions) ([]SyncResult, error)
```

### 5. Drift Detector

**Package:** `pkg/drift`

**Responsibilities:**
- Periodic state comparison
- Drift scoring
- Notification
- Optional auto-healing

**Data Structures:**

```go
package drift

import (
    "github.com/helmfile/helmfile/pkg/state"
)

// Detector monitors for configuration drift
type Detector struct {
    helmState      *state.HelmState
    interval       time.Duration
    autoHeal       bool
    notifiers      []Notifier
    ctx            context.Context
    cancel         context.CancelFunc
}

// DriftReport describes detected drift
type DriftReport struct {
    Timestamp      time.Time
    ReleaseName    string
    Namespace      string
    DriftType      DriftType
    Severity       Severity
    Details        string
    Diff           string
    Healed         bool
}

type DriftType string

const (
    DriftTypeConfiguration DriftType = "configuration"
    DriftTypeResource      DriftType = "resource"
    DriftTypeImage         DriftType = "image"
    DriftTypeDeletion      DriftType = "deletion"
)

type Severity string

const (
    SeverityLow    Severity = "low"
    SeverityMedium Severity = "medium"
    SeverityHigh   Severity = "high"
)

type Notifier interface {
    Notify(report DriftReport) error
}

func NewDetector(helmState *state.HelmState, interval time.Duration) *Detector
func (d *Detector) Start(ctx context.Context) error
func (d *Detector) Stop() error
func (d *Detector) AddNotifier(n Notifier)
func (d *Detector) EnableAutoHeal(enable bool)
```

### 6. Daemon Manager

**Package:** `pkg/daemon`

**Responsibilities:**
- Background process management
- PID file handling
- API server (HTTP or Unix socket)
- Health checks
- Graceful shutdown

**Data Structures:**

```go
package daemon

// Daemon manages background helmfire process
type Daemon struct {
    pidFile     string
    socketPath  string
    apiServer   *APIServer
    shutdownCh  chan os.Signal
    healthCh    chan bool
}

// APIServer provides control interface
type APIServer struct {
    addr    string
    handler *APIHandler
}

// APIHandler handles control commands
type APIHandler struct {
    substitutor  *substitute.Manager
    watcher      *watcher.Watcher
    detector     *drift.Detector
}

func NewDaemon(pidFile, socketPath string) *Daemon
func (d *Daemon) Start() error
func (d *Daemon) Stop() error
func (d *Daemon) IsRunning() (bool, error)
func (d *Daemon) GetPID() (int, error)
```

**API Endpoints:**
```
GET  /health              - Health check
GET  /status              - Current status
POST /chart/add           - Add chart substitution
POST /chart/remove        - Remove chart substitution
POST /image/add           - Add image substitution
POST /image/remove        - Remove image substitution
GET  /substitutions       - List all substitutions
POST /sync                - Trigger manual sync
GET  /drift               - Get drift reports
POST /reload              - Reload helmfile state
POST /shutdown            - Graceful shutdown
```

---

## Reusable Libraries

### From Helmfile

| Package | Purpose | Import Path |
|---------|---------|-------------|
| `pkg/state` | State management, release planning, sync execution | `github.com/helmfile/helmfile/pkg/state` |
| `pkg/app` | Application orchestration, helmfile loading | `github.com/helmfile/helmfile/pkg/app` |
| `pkg/helmexec` | Helm command execution interface | `github.com/helmfile/helmfile/pkg/helmexec` |
| `pkg/event` | Hook system for lifecycle events | `github.com/helmfile/helmfile/pkg/event` |
| `pkg/environment` | Environment variable management | `github.com/helmfile/helmfile/pkg/environment` |
| `pkg/yaml` | YAML encoding/decoding utilities | `github.com/helmfile/helmfile/pkg/yaml` |

### From Helm

| Package | Purpose | Import Path |
|---------|---------|-------------|
| `pkg/chart/v2/loader` | Chart loading from disk/archives | `helm.sh/helm/v4/pkg/chart/v2/loader` |
| `pkg/engine` | Template rendering with Sprig | `helm.sh/helm/v4/pkg/engine` |
| `pkg/cli/values` | Values file parsing and merging | `helm.sh/helm/v4/pkg/cli/values` |
| `pkg/chart/common/util` | Chart utilities, value coalescing | `helm.sh/helm/v4/pkg/chart/common/util` |
| `pkg/kube` | Kubernetes client, resource management | `helm.sh/helm/v4/pkg/kube` |
| `pkg/storage` | Release storage backends | `helm.sh/helm/v4/pkg/storage` |
| `pkg/action` | Release action framework | `helm.sh/helm/v4/pkg/action` |
| `pkg/postrenderer` | Post-render pipeline | `helm.sh/helm/v4/pkg/postrenderer` |

### Third-Party Libraries

| Library | Purpose | Import Path |
|---------|---------|-------------|
| fsnotify | File system watching | `github.com/fsnotify/fsnotify` |
| cobra | CLI framework | `github.com/spf13/cobra` |
| viper | Configuration management | `github.com/spf13/viper` |
| zap | Structured logging | `go.uber.org/zap` |
| yaml.v3 | YAML processing | `gopkg.in/yaml.v3` |
| client-go | Kubernetes API client | `k8s.io/client-go` |

---

## Data Flow

### 1. Initial Sync Flow

```
User: helmfire sync --watch

1. Load Configuration
   ├─ Parse CLI flags
   ├─ Load helmfile.yaml
   └─ Initialize logger

2. Initialize Components
   ├─ Create Substitution Manager (empty)
   ├─ Create Helmfile State Manager
   ├─ Create Sync Orchestrator
   ├─ Create File Watcher
   └─ Create Drift Detector (if enabled)

3. Load Helmfile State
   ├─ Parse helmfile.yaml
   ├─ Resolve environment
   ├─ Load values files
   └─ Build release DAG

4. Apply Substitutions
   ├─ Apply chart substitutions (none initially)
   └─ Apply image substitutions (none initially)

5. Execute Sync
   ├─ Sync repositories
   ├─ Prepare charts
   ├─ For each release batch (parallel):
   │  ├─ Pre-sync hooks
   │  ├─ Helm upgrade --install
   │  └─ Post-sync hooks
   └─ Report results

6. Start Watchers
   ├─ Register helmfile.yaml
   ├─ Register all values files
   ├─ Register local chart directories (if any)
   └─ Start fsnotify

7. Start Drift Detector (if enabled)
   └─ Schedule periodic diff checks

8. Wait for Events
   ├─ File change events → Selective sync
   ├─ Drift detection → Report/heal
   └─ Control commands → Execute
```

### 2. Chart Substitution Flow

```
User: helmfire chart bitnami/postgresql ./charts/postgresql

1. Validate Input
   ├─ Check local path exists
   ├─ Verify it's a valid chart
   └─ Load chart metadata

2. Register Substitution
   ├─ Add to Substitution Manager
   └─ Add chart directory to File Watcher

3. Reload State
   ├─ Re-parse helmfile.yaml
   └─ Apply chart substitution

4. Identify Affected Releases
   └─ Find releases using bitnami/postgresql

5. Selective Sync
   ├─ Sync only affected releases
   └─ Report results

6. Monitor Local Chart
   └─ Watch for changes in ./charts/postgresql
```

### 3. File Change Flow

```
File System Event: helmfile.yaml modified

1. Debounce
   ├─ Wait 500ms for more changes
   └─ Collect all events in window

2. Determine Impact
   ├─ Parse changed file
   ├─ Compare with previous state
   └─ Identify affected releases

3. Reload State
   ├─ Re-parse helmfile.yaml
   └─ Apply current substitutions

4. Selective Sync
   ├─ Sync only affected releases
   ├─ Skip unchanged releases
   └─ Report results

5. Resume Watching
```

### 4. Drift Detection Flow

```
Timer: 30 seconds elapsed

1. Run Diff
   ├─ For each release:
   │  ├─ Get desired state (helmfile)
   │  ├─ Get actual state (cluster)
   │  └─ Compare

2. Score Drift
   ├─ Configuration changes
   ├─ Resource changes
   ├─ Image changes
   └─ Deletions

3. Generate Report
   ├─ Timestamp
   ├─ Affected releases
   ├─ Drift details
   └─ Severity

4. Notify
   ├─ Log to stdout
   ├─ Send webhook (if configured)
   └─ Write to file (if configured)

5. Auto-Heal (if enabled)
   ├─ Sync affected releases
   └─ Update report

6. Schedule Next Check
```

---

## API Design

### Command-Line Interface

```bash
# Main sync command with watching
helmfire sync [flags]

Flags:
  --watch                     Watch for changes and auto-sync
  --daemon                    Run as background daemon
  --drift-detect              Enable drift detection
  --drift-interval duration   Drift check interval (default 30s)
  --drift-auto-heal           Automatically heal drift
  -f, --file string           Helmfile path (default helmfile.yaml)
  -e, --environment string    Environment name
  -l, --selector string       Label selector
  --concurrency int           Parallel execution limit (default 0)
  --interactive               Prompt before each release

# Chart substitution
helmfire chart <original> <local-path> [flags]

Examples:
  helmfire chart bitnami/postgresql ./charts/postgresql
  helmfire chart stable/mysql ../mysql-chart

Flags:
  --daemon-socket string      Daemon socket path (if daemon running)

# Image substitution
helmfire image <original> <replacement> [flags]

Examples:
  helmfire image postgres:15 localhost:5000/postgres:test
  helmfire image nginx:1.21 myregistry.io/nginx:custom

Flags:
  --daemon-socket string      Daemon socket path (if daemon running)

# Daemon control
helmfire daemon start [flags]
helmfire daemon stop [flags]
helmfire daemon status [flags]
helmfire daemon logs [flags]

# List substitutions
helmfire list charts
helmfire list images

# Remove substitutions
helmfire remove chart <original>
helmfire remove image <original>
```

### Daemon API (HTTP)

```http
# Add chart substitution
POST /api/v1/charts
Content-Type: application/json

{
  "original": "bitnami/postgresql",
  "localPath": "./charts/postgresql"
}

# Add image substitution
POST /api/v1/images
Content-Type: application/json

{
  "original": "postgres:15",
  "replacement": "localhost:5000/postgres:test"
}

# List substitutions
GET /api/v1/substitutions

Response:
{
  "charts": [
    {"original": "bitnami/postgresql", "localPath": "./charts/postgresql"}
  ],
  "images": [
    {"original": "postgres:15", "replacement": "localhost:5000/postgres:test"}
  ]
}

# Trigger manual sync
POST /api/v1/sync
Content-Type: application/json

{
  "releases": ["app1", "app2"],  // optional, empty = all
  "dryRun": false
}

# Get drift reports
GET /api/v1/drift?since=2024-01-01T00:00:00Z

# Health check
GET /health

# Status
GET /api/v1/status

Response:
{
  "running": true,
  "uptime": "2h30m",
  "lastSync": "2024-01-15T10:30:00Z",
  "watchedFiles": 15,
  "activeSubstitutions": {
    "charts": 2,
    "images": 3
  }
}
```

---

## Implementation Phases

### Phase 1: Foundation (Week 1-2)

**Goal:** Basic helmfile sync with substitution manager

**Tasks:**
1. Project setup
   - Initialize Go module
   - Set up directory structure
   - Configure linters and CI

2. Core packages
   - Implement `pkg/substitute` (Manager)
   - Implement `pkg/helmstate` (wrapper around helmfile)
   - Implement `pkg/sync` (Orchestrator)

3. CLI basics
   - `helmfire sync` command (no watching)
   - `helmfire chart` command
   - `helmfire image` command

4. Testing
   - Unit tests for Substitution Manager
   - Integration test with sample helmfile

**Deliverables:**
- `helmfire sync` works like `helmfile sync`
- `helmfire chart <orig> <local>` substitutes chart
- `helmfire image <orig> <new>` substitutes image

### Phase 2: File Watching (Week 3-4)

**Goal:** Auto-reload on file changes

**Tasks:**
1. Implement `pkg/watcher`
   - fsnotify integration
   - Debouncing logic
   - Impact analysis

2. Enhance `helmfire sync`
   - Add `--watch` flag
   - Integrate File Watcher
   - Selective sync on changes

3. Testing
   - Test file watching with various change scenarios
   - Test debouncing
   - Test selective sync

**Deliverables:**
- `helmfire sync --watch` auto-reloads on file changes
- Changes trigger only affected releases
- Debouncing prevents rapid re-syncs

### Phase 3: Drift Detection (Week 5)

**Goal:** Monitor cluster state vs. desired state

**Tasks:**
1. Implement `pkg/drift`
   - Drift Detector
   - Notifier interface
   - Stdout notifier
   - Webhook notifier

2. Enhance `helmfire sync`
   - Add `--drift-detect` flag
   - Add `--drift-interval` flag
   - Add `--drift-auto-heal` flag

3. Testing
   - Test drift detection with manual changes
   - Test auto-healing
   - Test notifications

**Deliverables:**
- `helmfire sync --drift-detect` detects configuration drift
- Drift reports generated
- Auto-heal option works

### Phase 4: Daemon Mode (Week 6)

**Goal:** Background process with API control

**Tasks:**
1. Implement `pkg/daemon`
   - Daemon process management
   - PID file handling
   - HTTP API server
   - Unix socket support

2. Add daemon commands
   - `helmfire daemon start`
   - `helmfire daemon stop`
   - `helmfire daemon status`
   - `helmfire daemon logs`

3. Update control commands
   - `helmfire chart` sends to daemon API
   - `helmfire image` sends to daemon API

4. Testing
   - Test daemon lifecycle
   - Test API endpoints
   - Test graceful shutdown

**Deliverables:**
- `helmfire sync --daemon` runs in background
- API control via HTTP
- Daemon survives terminal closure

### Phase 5: Polish & Documentation (Week 7-8)

**Goal:** Production-ready tool

**Tasks:**
1. Comprehensive testing
   - End-to-end tests
   - Performance tests
   - Error handling coverage

2. Documentation
   - README with examples
   - Architecture docs
   - API reference
   - Contribution guide

3. Tooling
   - Release automation
   - Homebrew formula
   - Docker image

4. Examples
   - Sample helmfile projects
   - Common use cases
   - Tutorial videos

**Deliverables:**
- Stable v1.0.0 release
- Complete documentation
- Distribution packages

---

## Technical Decisions

### 1. Why Go?

**Decision:** Implement helmfire in Go

**Rationale:**
- Helmfile and Helm are written in Go
- Can import their packages directly as libraries
- Excellent concurrency primitives for watching/drift detection
- Single binary distribution
- Strong Kubernetes ecosystem support

### 2. Library vs. Fork

**Decision:** Import helmfile and helm as libraries, not fork

**Rationale:**
- Benefit from upstream improvements and bug fixes
- Smaller codebase to maintain
- Clear separation of concerns
- Easier to update dependencies

### 3. State Storage

**Decision:** Use in-memory state, rely on helmfile/helm for persistence

**Rationale:**
- Helmfire is ephemeral (runs, watches, exits)
- State of releases stored by helm in Kubernetes Secrets
- Helmfile.yaml is source of truth
- No need for additional storage layer

### 4. File Watching Library

**Decision:** Use `github.com/fsnotify/fsnotify`

**Rationale:**
- Cross-platform (Linux, macOS, Windows)
- Well-maintained, widely used
- Native OS event support (inotify, FSEvents, etc.)
- Stable API

### 5. API Interface

**Decision:** HTTP API for daemon control (not gRPC)

**Rationale:**
- Simpler for end users (curl works)
- No protobuf compilation needed
- REST is familiar
- Good enough for control plane traffic

### 6. Configuration Format

**Decision:** Reuse helmfile.yaml, no helmfire-specific config file

**Rationale:**
- Helmfire is transparent to helmfile
- Drop-in replacement workflow
- Substitutions are runtime, not config
- Reduces complexity

### 7. Image Substitution Method

**Decision:** Post-renderer approach (not pre-template)

**Rationale:**
- Templates already rendered correctly
- Clean separation: templates work, then substitute
- Can substitute in charts we don't control
- Consistent with Helm's extensibility model

### 8. Drift Detection Trigger

**Decision:** Periodic polling (not Kubernetes watch)

**Rationale:**
- Simpler implementation
- Kubernetes watch requires per-resource watches (complex)
- Polling every 30s is acceptable overhead
- Aligns with helmfile diff workflow

### 9. Concurrency Model

**Decision:** Inherit helmfile's concurrency model

**Rationale:**
- Proven in production
- Configurable worker pool
- DAG-based release ordering
- Reuse existing code

### 10. Logging

**Decision:** Use structured logging with `go.uber.org/zap`

**Rationale:**
- Performance (structured is faster)
- Machine-readable output
- Already used by helmfile
- Supports multiple output formats

---

## Security Considerations

### 1. Local Chart Path Validation

**Risk:** Path traversal attacks

**Mitigation:**
- Validate all paths are within allowed directories
- Resolve symlinks and check final path
- Use `filepath.Clean()` and `filepath.Abs()`

### 2. Image Substitution

**Risk:** Unintended registry access

**Mitigation:**
- Validate image references (must be well-formed)
- Log all substitutions
- Require explicit user action (no auto-substitution)

### 3. Daemon API

**Risk:** Unauthorized access to control API

**Mitigation:**
- Unix socket with file permissions (default)
- HTTP with localhost binding only
- Optional authentication token
- TLS support for remote access

### 4. File Watching

**Risk:** Resource exhaustion from watching too many files

**Mitigation:**
- Limit max watched files (configurable)
- Warn on excessive watchers
- Graceful degradation

### 5. Drift Auto-Heal

**Risk:** Accidental revert of intentional changes

**Mitigation:**
- Auto-heal disabled by default
- Require explicit flag
- Log all healing actions
- Dry-run mode for testing

---

## Performance Considerations

### 1. File Watching Overhead

**Challenge:** Too many file watchers

**Solution:**
- Watch parent directories, filter events
- Debouncing reduces sync frequency
- Configurable debounce duration

### 2. Drift Detection Load

**Challenge:** Frequent diffs are expensive

**Solution:**
- Configurable interval (default 30s)
- Can disable drift detection
- Selective diff (only changed releases)

### 3. Selective Sync

**Challenge:** Determining affected releases

**Solution:**
- Build dependency graph
- Track file → release mappings
- Cache analysis results

### 4. Memory Usage

**Challenge:** Long-running daemon

**Solution:**
- Release state after sync (GC can reclaim)
- Limit drift report history
- Stream logs rather than buffer

### 5. Startup Time

**Challenge:** Initial sync can be slow

**Solution:**
- Parallel chart downloads (inherited from helmfile)
- Concurrent release execution
- Progress reporting for user feedback

---

## Future Enhancements

### Post-MVP Features

1. **GUI Dashboard**
   - Web UI for monitoring
   - Visualize drift over time
   - Control substitutions
   - View logs

2. **Multi-Environment Support**
   - Watch multiple environments simultaneously
   - Per-environment substitutions
   - Environment comparison

3. **Advanced Notifications**
   - Slack integration
   - Email alerts
   - Webhook templates
   - PagerDuty integration

4. **Smart Healing**
   - ML-based drift classification
   - Intelligent healing decisions
   - Rollback on failed heal

5. **Cluster-Wide Monitoring**
   - Monitor all releases (not just helmfile-managed)
   - Namespace watching
   - Resource usage tracking

6. **Plugin System**
   - Custom substitution strategies
   - Custom drift detectors
   - Custom notifiers

7. **Configuration Backup**
   - Git-based backup of helmfile state
   - Automatic commits on change
   - Rollback support

8. **Performance Optimizations**
   - Incremental diff (only changed resources)
   - Caching of chart metadata
   - Parallel drift detection

---

## Conclusion

Helmfire extends helmfile with developer-friendly features:
- **Watch mode** for instant feedback
- **Live substitution** for rapid iteration
- **Drift detection** for production monitoring
- **Daemon mode** for continuous operations

By reusing helmfile and helm libraries, helmfire provides these features with minimal new code and maximum compatibility.

**Next Steps:**
1. Initialize Go project
2. Implement Phase 1 components
3. Create integration tests
4. Build and iterate

---

**Document Version:** 1.0
**Last Updated:** 2025-11-15
**Authors:** Helmfire Team
**Status:** Design Approved

# Helm Project Architecture Analysis

## Executive Summary

The Helm project is a sophisticated Kubernetes package manager written in Go. It is organized as a modular system with clear separation between CLI commands (`pkg/cmd`), core actions (`pkg/action`), chart management (`pkg/chart`), Kubernetes integration (`pkg/kube`), and storage backends (`pkg/storage`). The project uses semantic versioning (v1, v2, v3) for chart formats and provides extensibility through plugins and post-renderers.

**Key Technologies:**
- Go 1.24.0
- Kubernetes client-go v0.34.2
- ORAS v2 for OCI registry support
- Cobra for CLI framework
- Sprig v3 for template functions
- JSON Schema v6 for values validation

---

## 1. PROJECT STRUCTURE

### 1.1 Top-Level Organization

```
/home/user/helmfire/analysis/sources/helm/
├── cmd/helm/                    # Main executable entry point
├── pkg/                         # Public reusable packages (primary library)
├── internal/                    # Internal packages (not exported)
├── testdata/                    # Test fixtures
└── scripts/                     # Build and utility scripts
```

### 1.2 Package Hierarchy

#### Public Packages (`pkg/`)
- **action/** - Release lifecycle operations (install, upgrade, rollback, uninstall)
- **chart/** - Chart abstraction layer and loading
- **kube/** - Kubernetes API client integration
- **storage/** - Release history storage
- **engine/** - Template rendering engine
- **cli/** - CLI utilities and configuration
- **cmd/** - Cobra command implementations
- **repo/** - Chart repository management
- **registry/** - OCI registry client
- **downloader/** - Dependency resolution and downloading
- **postrenderer/** - Post-render pipeline plugins
- **getter/** - HTTP/S3/GCS file retrieval
- **helmpath/** - Path management for Helm homes
- **release/** - Release data structures
- **strvals/** - String value parsing utilities

#### Internal Packages (`internal/`)
- **chart/** - Chart v3 implementation
- **cli/** - CLI infrastructure
- **logging/** - Logging utilities
- **plugin/** - Plugin system infrastructure
- **resolver/** - Dependency resolver
- **statusreaders/** - Kubernetes status readers
- **tlsutil/** - TLS utilities
- **urlutil/** - URL utilities
- **version/** - Version information

---

## 2. CORE COMPONENTS

### 2.1 Chart Management and Loading

#### Chart Data Structures

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/chart/v2/chart.go` (lines 36-64)

```go
type Chart struct {
    Raw       []*common.File              // Raw chart files
    Metadata  *Metadata                   // Chart.yaml metadata
    Lock      *Lock                       // Chart.lock dependencies
    Templates []*common.File              // Template files
    Values    map[string]interface{}      // Default values
    Schema    []byte                      // JSON schema for values
    Files     []*common.File              // Misc files (README, LICENSE)
    parent    *Chart                      // Parent chart reference
    dependencies []*Chart                 // Subchart dependencies
}
```

**Chart Loader Architecture**

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/chart/v2/loader/load.go` (lines 40-69)

The loader uses a ChartLoader interface with two implementations:
- `FileLoader` - Loads from `.tgz` archives via the `archive` package
- `DirLoader` - Loads from directories, respects `.helmignore`

Key loading process:
1. Detects file type (archive vs directory)
2. Parses Chart.yaml into Metadata
3. Loads templates from `templates/` directory
4. Processes values.yaml
5. Loads CRDs from `crds/` directory
6. Handles subchart dependencies from `charts/` directory

**Key Functions:**
- `Load(name string) (*Chart, error)` - Main entry point
- `LoadFiles(files []*archive.BufferedFile) (*Chart, error)` - In-memory loading

#### Chart Accessor Pattern

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/chart/interfaces.go` (lines 26-45)

Helm uses an abstraction pattern to support multiple chart versions (v1, v2, v3):

```go
type Accessor interface {
    Name() string
    IsRoot() bool
    MetadataAsMap() map[string]interface{}
    Files() []*common.File
    Templates() []*common.File
    ChartFullPath() string
    IsLibraryChart() bool
    Dependencies() []Charter
    MetaDependencies() []Dependency
    Values() map[string]interface{}
    Schema() []byte
    Deprecated() bool
}
```

**Implementation:** `pkg/chart/common.go` provides:
- `v2Accessor` - For Helm v2/v3 charts
- `v3Accessor` - For v3 charts (via internal/chart/v3)

### 2.2 Values Handling and Merging

#### Values Merging Pipeline

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/cli/values/options.go` (lines 33-80)

Values are merged in this priority order:
1. Chart default values (`values.yaml`)
2. Values from `-f/--values` files (in order, later overrides earlier)
3. `--set-json` values
4. `--set` values
5. `--set-string` values
6. `--set-file` values
7. `--set-literal` values

**Values Options Structure:**
```go
type Options struct {
    ValueFiles    []string // -f/--values
    StringValues  []string // --set-string
    Values        []string // --set
    FileValues    []string // --set-file
    JSONValues    []string // --set-json
    LiteralValues []string // --set-literal
}

// MergeValues implements the merging logic
func (opts *Options) MergeValues(p getter.Providers) (map[string]interface{}, error)
```

#### Values Rendering

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/chart/common/util/values.go` (lines 26-70)

The `ToRenderValues` function composes final render values:

```go
type ReleaseOptions struct {
    Name      string
    Namespace string
    IsUpgrade bool
    IsInstall bool
    Revision  int
}

type Capabilities struct {
    APIVersions VersionSet
    KubeVersion KubeVersion
    HelmVersion Version
}

// Final render values structure:
top := map[string]interface{}{
    "Chart":        chartMetadata,
    "Capabilities": kubeCapabilities,
    "Release":      releaseOptions,
    "Values":       mergedValues,
}
```

#### Coalescing and Schema Validation

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/chart/common/util/coalesce.go`

- `CoalesceValues(chrt, vals)` - Merges chart defaults with provided values
- `ValidateAgainstSchema(chrt, vals)` - Validates using JSON Schema v6

### 2.3 Template Rendering Engine

#### Engine Architecture

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/engine/engine.go` (lines 37-82)

```go
type Engine struct {
    Strict              bool                  // Fail on undefined values
    LintMode            bool                  // Allow missing required values
    clientProvider      *ClientProvider       // K8s client for template funcs
    EnableDNS           bool                  // Allow DNS lookups
    CustomTemplateFuncs template.FuncMap      // User-defined functions
}

// Main rendering method
func (e Engine) Render(chrt Charter, values Values) (map[string]string, error)
```

**Rendering Process:**
1. Collects all template files from chart and subcharts
2. Creates a single template with all files
3. Executes templates with merged values
4. Returns map of filename → rendered YAML

**Template Functions:**
- Built-in Sprig functions (v3)
- Kubernetes-aware functions (from `pkg/engine/funcs.go`)
- Lookup functions for existing resources (when client provided)
- DNS lookup support (optional)
- Custom user-defined functions

#### Value Scoping for Subcharts

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/engine/engine.go` (lines 60-82)

Values are scoped to subchart dependencies:
- Parent chart cannot access subchart values
- Subchart can access its own values and parent's top-level values
- Hierarchical value passing based on chart structure

### 2.4 Release Lifecycle Management

#### Release Data Model

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/release/interfaces.go` (lines 25-41)

```go
type Accessor interface {
    Name() string
    Namespace() string
    Version() int           // Release revision number
    Hooks() []Hook
    Manifest() string       // Rendered YAML
    Notes() string          // Post-deployment notes
    Labels() map[string]string
    Chart() Charter
    Status() string         // deployed, failed, pending-upgrade, etc.
    ApplyMethod() string    // apply-method (ClientSideApply or ServerSideApply)
    DeployedAt() time.Time
}
```

#### Action Configuration

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/action/action.go` (lines 91-118)

```go
type Configuration struct {
    RESTClientGetter    RESTClientGetter      // K8s config loader
    Releases            *storage.Storage      // Release storage backend
    KubeClient          kube.Interface        // K8s API client
    RegistryClient      *registry.Client      // OCI registry client
    Capabilities        *common.Capabilities  // Cluster capabilities
    CustomTemplateFuncs template.FuncMap      // Custom template functions
    HookOutputFunc      func(ns, pod, container string) io.Writer
    mutex               sync.Mutex
}
```

All action implementations (Install, Upgrade, Rollback, etc.) depend on this Configuration.

#### Dry Run Strategy

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/action/action.go` (lines 75-88)

```go
type DryRunStrategy string

const (
    DryRunNone   DryRunStrategy = "none"    // Execute operation
    DryRunClient DryRunStrategy = "client"  // Client-side simulation
    DryRunServer DryRunStrategy = "server"  // Server-side dry-run
)
```

### 2.5 Key Action Implementations

#### Install

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/action/install.go` (lines 73-100)

```go
type Install struct {
    cfg            *Configuration
    ChartPathOptions              // Chart location options
    ForceReplace   bool           // Ignore warnings
    ForceConflicts bool           // Force SSA conflicts
    ServerSideApply bool          // Use SSA
    CreateNamespace bool
    DryRunStrategy DryRunStrategy
    HideSecret     bool
    DisableHooks   bool
    Replace        bool           // Allow reinstall
    WaitStrategy   kube.WaitStrategy
    WaitForJobs    bool
    Timeout        time.Duration
}
```

**Process:**
1. Loads chart from path/registry/repository
2. Merges values
3. Renders templates with engine
4. Runs pre-install hooks
5. Creates Kubernetes resources (with ServerSideApply)
6. Waits for resource readiness
7. Runs post-install hooks
8. Stores release in configured backend

#### Upgrade

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/action/upgrade.go` (lines 47-80)

Similar to Install with additional capabilities:
- Keeps previous values if not overridden
- Handles release version incrementing
- Supports rollback on failure
- Cleanup on failure option

#### Rollback

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/action/rollback.go` (lines 33-60)

```go
type Rollback struct {
    cfg               *Configuration
    Version           int         // Target revision
    Timeout           time.Duration
    WaitStrategy      kube.WaitStrategy
    DisableHooks      bool
    DryRunStrategy    DryRunStrategy
    ForceReplace      bool
    ServerSideApply   string      // "true", "false", or "auto"
    CleanupOnFail     bool
    MaxHistory        int
}
```

**Process:**
1. Retrieves target release revision
2. Renders previous manifest
3. Applies to cluster
4. Updates release status
5. Prunes old revisions if MaxHistory exceeded

#### Uninstall

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/action/action.go` (referenced in cmd/)

- Deletes resources from manifest
- Runs pre/post-delete hooks
- Preserves release record (by default)
- Supports cascading deletion policies

---

## 3. KUBERNETES CLIENT INTEGRATION

### 3.1 Kubernetes Client Interface

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/kube/interface.go` (lines 28-77)

```go
type Interface interface {
    Get(resources ResourceList, related bool) (map[string][]runtime.Object, error)
    Create(resources ResourceList, options ...ClientCreateOption) (*Result, error)
    Delete(resources ResourceList, policy metav1.DeletionPropagation) (*Result, []error)
    Update(original, target ResourceList, options ...ClientUpdateOption) (*Result, error)
    Build(reader io.Reader, validate bool) (ResourceList, error)
    BuildTable(reader io.Reader, validate bool) (ResourceList, error)
    IsReachable() error
    GetWaiter(ws WaitStrategy) (Waiter, error)
    GetPodList(namespace string, listOptions metav1.ListOptions) (*v1.PodList, error)
    OutputContainerLogsForPodList(podList *v1.PodList, namespace string, ...) error
}
```

### 3.2 Resource Management

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/kube/client.go` (lines 73-100+)

**Client Implementation:**
```go
type Client struct {
    Factory       // kubectl Factory interface
    // Other config fields
}
```

**Key Capabilities:**
- **Server-Side Apply (SSA)** - Kubernetes 1.18+ feature
  - Uses `application/apply-patch+json`
  - Handles field manager conflicts
  - ForceConflicts option: "Overwrite value, become sole manager"

- **Patching Strategies:**
  - Strategic Merge Patch (for native types)
  - JSON Merge Patch
  - JSON Patch (RFC 6902)

- **Resource Building:**
  - Parses YAML/JSON manifests
  - Validates against OpenAPI schema (optional)
  - Handles CRD instances

### 3.3 Wait/Ready Checking

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/kube/wait.go` (lines 47-78)

```go
type legacyWaiter struct {
    c          ReadyChecker
    kubeClient *kubernetes.Clientset
    ctx        context.Context
}

// Two wait methods:
func (hw *legacyWaiter) Wait(resources ResourceList, timeout time.Duration) error
func (hw *legacyWaiter) WaitWithJobs(resources ResourceList, timeout time.Duration) error
```

**Monitored Resource Types:**
- Deployments
- StatefulSets
- DaemonSets
- Jobs (optional)
- Pods
- Services
- PersistentVolumeClaims

**Wait Strategy:** Polls every 2 seconds until all resources ready or timeout.

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/kube/ready.go`

ReadyChecker evaluates readiness based on resource type:
- Pods: Running with all containers ready
- Deployments: All replicas ready and updated
- Services: Has endpoints (or is ExternalName)
- PVCs: Bound
- Jobs: Completed successfully

---

## 4. STORAGE BACKENDS

### 4.1 Storage Abstraction

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/storage/storage.go` (lines 40-100)

```go
type Storage struct {
    driver.Driver
    MaxHistory int  // Limit release revisions
}

// Main methods:
func (s *Storage) Get(name string, version int) (release.Releaser, error)
func (s *Storage) Create(rls release.Releaser) error
func (s *Storage) Update(rls release.Releaser) error
func (s *Storage) Delete(name string, version int) (release.Releaser, error)
func (s *Storage) ListReleases() ([]release.Releaser, error)
func (s *Storage) ListReleasesByStateMask(mask release.Status) ([]release.Releaser, error)
```

### 4.2 Available Drivers

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/storage/driver/`

**Drivers Available:**
1. **Secrets Driver** (`secrets.go`)
   - Stores releases as Kubernetes Secrets
   - Namespace-scoped
   - Default in production

2. **ConfigMaps Driver** (`cfgmaps.go`)
   - Stores releases as Kubernetes ConfigMaps
   - Namespace-scoped
   - Simpler, human-readable

3. **Memory Driver** (`memory.go`)
   - In-memory storage
   - Used for testing
   - Per-process, lost on restart

4. **SQL Driver** (`sql.go`)
   - PostgreSQL support
   - External storage for multi-cluster scenarios
   - Connection string via env: `HELM_DRIVER_SQL_CONNECTION_STRING`

### 4.3 Driver Interface

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/storage/driver/driver.go`

```go
type Driver interface {
    Create(key string, rls Releaser) error
    Get(key string) (Releaser, error)
    Update(key string, rls Releaser) error
    Delete(key string) (Releaser, error)
    List() ([]Releaser, error)
    Query(labels.Set) ([]Releaser, error)
}
```

**Release Metadata:**
- Storage key: `sh.helm.release.v1/{release-name}/{revision}`
- Labels: namespace, owner, status, version
- Compression: Gzip (transparent)

---

## 5. CHART REPOSITORY AND REGISTRY SUPPORT

### 5.1 Chart Repository Management

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/repo/v1/`

Helm supports traditional chart repositories (HTTP/S):
- **Index.yaml** - Machine-readable chart listing
- **ChartRepo** - Manages repo metadata
- **RepositoryIndex** - Parses and queries index
- URL-based chart references: `{repo-name}/{chart-name}`

### 5.2 OCI Registry Support

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/registry/client.go` (lines 56-78)

```go
type Client struct {
    debug              bool
    enableCache        bool
    credentialsFile    string      // ~/.docker/config.json
    username           string
    password           string
    authorizer         *auth.Client    // ORAS auth
    registryAuthorizer RemoteClient    // HTTP client
    credentialsStore   credentials.Store
    httpClient         *http.Client
    plainHTTP          bool
}
```

**OCI Integration:**
- Based on **ORAS v2** (OCI Registry as Storage)
- Stores charts as OCI artifacts
- Container registry compatible (Docker Hub, ECR, GCR, etc.)
- Chart reference: `{registry}/{organization}/{chart}:{version}`
- Version tag translation: `+` → `_` (OCI limitation)

**Push/Pull Operations:**
- `Push(ctx, ref string, chart *Chart, opts ...ClientOption) (digest, error)`
- `Pull(ctx, ref string, opts ...ClientOption) (*Chart, error)`
- `Login(ctx, hostname string, opts ...ClientOption) error`
- `Logout(ctx, hostname string) error`

### 5.3 Dependency Downloader

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/downloader/manager.go` (lines 59-80)

```go
type Manager struct {
    Out                io.Writer
    ChartPath          string          // Chart directory
    Verify             VerificationStrategy
    Debug              bool
    Keyring            string          // GPG keyring
    SkipUpdate         bool
    Getters            []getter.Provider
    RegistryClient     *registry.Client
    RepositoryConfig   string
    RepositoryCache    string
    ContentCache       string          // Dependency cache
}

// Main methods:
func (m *Manager) Update() error
func (m *Manager) Build() error
```

**Process:**
1. Parses Chart.yaml dependencies
2. Resolves chart locations (repo, OCI, local path)
3. Downloads charts to `charts/` directory
4. Verifies chart signatures (if enabled)
5. Resolves transitive dependencies
6. Supports semver version constraints

---

## 6. POST-RENDERING PIPELINE

### 6.1 Post Renderer Interface

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/postrenderer/postrenderer.go` (lines 28-35)

```go
type PostRenderer interface {
    Run(renderedManifests *bytes.Buffer) (modifiedManifests *bytes.Buffer, err error)
}
```

**Use Cases:**
- Kustomize post-processing
- Custom manifest modifications
- Annotation/label injection
- Security policy enforcement

### 6.2 Plugin-Based Post Renderers

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/postrenderer/postrenderer.go` (lines 38-70)

Post-renderers are plugins:
- Located in `$HELM_PLUGINS_DIR`
- Protocol: `postrenderer/v1`
- Communicate via stdin/stdout
- Run in separate process

**Integration in Rendering:**

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/action/action.go` (lines 285-310)

```go
if pr != nil {
    // Merge files into single YAML stream
    merged, err := annotateAndMerge(files)
    
    // Run post renderer
    postRendered, err := pr.Run(bytes.NewBufferString(merged))
    
    // Split back into per-file manifests
    files, err = splitAndDeannotate(postRendered.String())
}
```

---

## 7. COMMAND-LINE INTERFACE

### 7.1 Command Structure

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/cmd/root.go` (lines 17-100+)

Built with **Cobra** framework:

```
helm [global-flags] <command> [command-flags] [args]
```

**Global Flags:**
- `--debug` - Enable debug logging
- `--kubeconfig` - K8s config file
- `--kubecontext` - K8s context name
- `--namespace` - Target namespace
- `--registry-config` - Registry credentials file
- `--repository-cache` - Repo index cache dir
- `--repository-config` - Repos config file
- `--plugins` - Plugins directory

**Major Commands:**
- **chart** - Chart operations (create, package, verify)
- **completion** - Shell completion
- **dependency** - Chart dependencies (list, build, update)
- **env** - Environment configuration
- **get** - Retrieve release info (values, manifest, notes, all)
- **help** - Help system
- **history** - Release history
- **install** - Install a chart
- **list** - List releases
- **plugin** - Manage plugins
- **pull** - Download charts
- **push** - Push charts to registry
- **registry** - Registry authentication
- **repo** - Manage chart repositories
- **rollback** - Rollback to previous release
- **search** - Search repositories/hub
- **show** - Display chart values/readme/chart metadata
- **status** - Show release status
- **template** - Render templates locally
- **uninstall** - Uninstall release
- **upgrade** - Upgrade release
- **verify** - Verify chart signatures
- **version** - Show Helm version

### 7.2 Values Options

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/cli/values/options.go`

CLI flags for providing values:
```bash
helm install RELEASE CHART \
  -f values1.yaml -f values2.yaml \     # File-based values
  --set key1=val1,key2=val2 \            # Direct values
  --set-string str=value \               # Force string type
  --set-file key=file.txt \              # Load from file
  --set-json obj='{"key":"value"}' \    # JSON values
  --set-literal lit='${VAR}'             # Literal with no expansion
```

---

## 8. KEY INTERFACES AND ABSTRACTIONS

### 8.1 Charter Interface

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/chart/interfaces.go` (lines 22-44)

Represents any chart type:
```go
type Charter interface{}  // Empty interface for chart polymorphism
type Dependency interface{}
type Accessor interface { ... }  // Concrete operations
```

**Implementations:**
- `v2chart.Chart` - Helm v2/v3 charts
- `v3chart.Chart` - Newer v3 format (in internal/)

### 8.2 Releaser Interface

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/release/interfaces.go`

```go
type Releaser interface{}  // Empty interface for release polymorphism
type Accessor interface { ... }  // Concrete operations
```

**Implementations:**
- `v1release.Release` - Primary implementation

### 8.3 Getter Providers

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/getter/`

Pluggable file retrieval:
- HTTP/HTTPS
- File (local)
- OCI (Registry)
- S3, GCS, Azure (via plugins)

---

## 9. REUSABLE COMPONENTS FOR HELMFIRE

### 9.1 Directly Importable Packages

These packages are designed as libraries and can be imported:

**1. Chart Loading and Processing**
```go
import (
    chart "helm.sh/helm/v4/pkg/chart/v2"
    "helm.sh/helm/v4/pkg/chart/v2/loader"
    chartutil "helm.sh/helm/v4/pkg/chart/v2/util"
    "helm.sh/helm/v4/pkg/chart/common"
)

// Load a chart
c, err := loader.Load("/path/to/chart")

// Access chart via accessor
accessor, _ := chart.NewAccessor(c)
values := accessor.Values()
```

**2. Template Rendering**
```go
import "helm.sh/helm/v4/pkg/engine"

// Render templates
files, err := engine.Render(chart, values)

// Or with client context for lookup functions
eng := engine.New(restConfig)
files, err := eng.Render(chart, values)
```

**3. Values Management**
```go
import (
    "helm.sh/helm/v4/pkg/chart/common/util"
    "helm.sh/helm/v4/pkg/cli/values"
)

// Parse and merge values
opts := &values.Options{
    ValueFiles: []string{"values.yaml"},
    Values: []string{"key=value"},
}
merged, err := opts.MergeValues(getters)

// Coalesce with chart defaults
vals, err := util.CoalesceValues(chart, userValues)

// Create render context
renderVals, err := util.ToRenderValues(chart, vals, releaseOpts, caps)
```

**4. Kubernetes Integration**
```go
import (
    "helm.sh/helm/v4/pkg/kube"
    "k8s.io/cli-runtime/pkg/genericclioptions"
)

// Create Kube client
getter := genericclioptions.NewConfigFlags(...)
kubeClient := kube.New(getter)

// Build resources from YAML
reader := bytes.NewBufferString(yamlManifest)
resources, err := kubeClient.Build(reader, true)

// Apply resources
result, err := kubeClient.Create(resources)

// Wait for readiness
waiter, _ := kubeClient.GetWaiter(kube.WaitForJobs)
err = waiter.Wait(resources, timeout)
```

**5. Release Storage**
```go
import (
    "helm.sh/helm/v4/pkg/storage"
    "helm.sh/helm/v4/pkg/storage/driver"
)

// Create storage with secrets backend
secretsDriver := driver.NewSecrets(secretClient)
store := storage.Init(secretsDriver)

// Store/retrieve releases
err := store.Create(release)
rel, err := store.Get(name, version)
releases, err := store.ListReleases()
```

**6. Action Framework**
```go
import "helm.sh/helm/v4/pkg/action"

// Create configuration
cfg := action.NewConfiguration()
cfg.Init(getter, namespace, "secret")

// Use actions
install := action.NewInstall(cfg)
install.Namespace = "default"
release, err := install.Run(chart, values)

upgrade := action.NewUpgrade(cfg)
release, err := upgrade.Run(name, chart, values)

rollback := action.NewRollback(cfg)
err := rollback.Run(name)
```

**7. Chart Repository**
```go
import "helm.sh/helm/v4/pkg/repo/v1"

// Manage repositories
repos, err := v1.LoadRepositoriesFile(file)
repo, err := repos.Get(repoName)

// Search and download
index, err := repo.DownloadIndexFile(cache)
entries := index.Search(query)
```

**8. OCI Registry**
```go
import "helm.sh/helm/v4/pkg/registry"

// Create registry client
regClient, err := registry.NewClient()

// Push/pull charts
digest, err := regClient.Push(ctx, ref, chart)
pulledChart, err := regClient.Pull(ctx, ref)

// Authentication
err := regClient.Login(ctx, hostname, ...)
err := regClient.Logout(ctx, hostname)
```

**9. Downloader**
```go
import "helm.sh/helm/v4/pkg/downloader"

// Manage dependencies
manager := &downloader.Manager{
    ChartPath: "/path/to/chart",
}
err := manager.Update()
err := manager.Build()
```

**10. Post Renderers**
```go
import "helm.sh/helm/v4/pkg/postrenderer"

// Create plugin-based post renderer
pr, err := postrenderer.NewPostRendererPlugin(
    settings,
    "my-postrenderer",
    "--arg1=value1",
)

// Or implement custom renderer
type MyRenderer struct {}
func (m *MyRenderer) Run(manifests *bytes.Buffer) (*bytes.Buffer, error) {
    // Custom processing
    return modifiedManifests, nil
}
```

### 9.2 Helm Data Structures

**Release Metadata:**
```go
type Release struct {
    Name      string
    Namespace string
    Version   int
    Manifest  string          // Rendered YAML
    Info      *Info           // Status info
    Chart     *Chart          // Chart object
    Values    map[string]interface{}
    Hooks     []*Hook
}
```

**Release Status Types:**
```
deployed
superseded
failed
uninstalling
uninstalled
pending-install
pending-upgrade
pending-rollback
```

**Hook Types:**
```
pre-install, post-install
pre-upgrade, post-upgrade
pre-rollback, post-rollback
pre-delete, post-delete
test
```

### 9.3 Configuration Management

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/cmd/root.go`

Environment Variables:
```bash
HELM_CACHE_HOME             # ~/.cache/helm
HELM_CONFIG_HOME            # ~/.config/helm
HELM_DATA_HOME              # ~/.local/share/helm
HELM_DEBUG                  # Enable debug logging
HELM_DRIVER                 # secret|configmap|memory|sql
HELM_DRIVER_SQL_CONNECTION_STRING
HELM_MAX_HISTORY            # Max release versions
HELM_NAMESPACE              # Default namespace
HELM_NO_PLUGINS             # Disable plugins
HELM_PLUGINS                # Plugins directory
HELM_REGISTRY_CONFIG        # Registry auth file
HELM_REPOSITORY_CACHE       # Repo cache dir
HELM_REPOSITORY_CONFIG      # Repos config file
```

---

## 10. DEPENDENCIES AND KEY LIBRARIES

### 10.1 Kubernetes Libraries
- **k8s.io/client-go v0.34.2** - Kubernetes client library
- **k8s.io/api v0.34.2** - Kubernetes API types
- **k8s.io/apimachinery v0.34.2** - Common types and utilities
- **k8s.io/cli-runtime v0.34.2** - kubectl runtime utilities
- **k8s.io/kubectl v0.34.2** - kubectl internals

### 10.2 OCI and Registry
- **oras.land/oras-go/v2 v2.6.0** - OCI artifact storage
- **github.com/opencontainers/image-spec v1.1.1** - OCI image specification
- **github.com/distribution/distribution/v3 v3.0.0** - Docker registry

### 10.3 Chart Processing
- **github.com/Masterminds/semver/v3 v3.4.0** - Semantic versioning
- **github.com/Masterminds/sprig/v3 v3.3.0** - Template functions
- **sigs.k8s.io/kustomize/kyaml v0.21.0** - YAML parsing/manipulation

### 10.4 CLI and Configuration
- **github.com/spf13/cobra v1.10.1** - CLI framework
- **github.com/spf13/pflag v1.0.10** - Flags
- **k8s.io/cli-runtime/pkg/genericclioptions** - kubectl flags

### 10.5 Data Processing
- **go.yaml.in/yaml/v3 v3.0.4** - YAML parsing
- **sigs.k8s.io/yaml v1.6.0** - Type-safe YAML
- **sigs.k8s.io/kustomize/api v0.20.1** - Kustomize integration
- **github.com/santhosh-tekuri/jsonschema/v6 v6.0.2** - JSON Schema validation

### 10.6 Cryptography and Security
- **golang.org/x/crypto v0.44.0** - Crypto operations
- **github.com/ProtonMail/go-crypto v1.3.0** - GPG support
- **github.com/cyphar/filepath-securejoin v0.6.0** - Secure path handling

### 10.7 Plugin System
- **github.com/extism/go-sdk v1.7.1** - Extism plugin runtime
- **github.com/tetratelabs/wazero v1.10.1** - WASM runtime

### 10.8 Utilities
- **github.com/Masterminds/squirrel v1.5.4** - SQL query builder
- **github.com/mitchellh/copystructure v1.2.0** - Struct copying
- **github.com/fatih/color v1.18.0** - Colored output
- **github.com/gosuri/uitable v0.0.4** - Table formatting

---

## 11. WATCH AND MONITORING CAPABILITIES

### 11.1 Status Checking

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/action/status.go`

The Status action queries current deployment state:
```go
func (s *Status) Run(name string) (ri.Releaser, error) {
    // Get last release
    rel, _ := s.cfg.releaseContent(name, s.Version)
    
    // Build resources from manifest
    resources, _ := s.cfg.KubeClient.Build(...)
    
    // Retrieve current state
    resp, _ := s.cfg.KubeClient.Get(resources, true)
    
    // Attach to release info
    rel.Info.Resources = resp
    
    return rel, nil
}
```

### 11.2 Resource Readiness Checking

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/kube/ready.go`

ReadyChecker monitors:
- Pod readiness conditions
- Deployment replica counts
- StatefulSet readiness
- DaemonSet readiness
- Job completion
- Service endpoints
- PVC binding

Polling interval: 2 seconds
Monitored conditions: `Ready`, `Available`, `Progressing`

### 11.3 Log Output from Hooks

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/action/action.go` (lines 110-112)

```go
type Configuration struct {
    HookOutputFunc func(namespace, pod, container string) io.Writer
}
```

Hooks can output logs to custom writers for monitoring.

---

## 12. IMAGE AND CHART REFERENCE HANDLING

### 12.1 Chart References

Helm supports multiple chart reference formats:

**Traditional Repository:**
```
stable/mysql
```

**Full Repository URL:**
```
https://example.com/charts/mysql
```

**OCI Registry:**
```
oci://registry.example.com/charts/mysql:1.0.0
docker.io/library/mysql:1.0.0
```

**File Paths:**
```
./local/chart
/absolute/path/to/chart
file:///path/to/chart.tgz
```

### 12.2 Version Constraints

**File:** `/home/user/helmfire/analysis/sources/helm/pkg/downloader/`

Supports semantic version constraints:
```
1.2.3          # Exact version
~1.2.3         # Tilde: >= 1.2.3, < 1.3.0
^1.2.3         # Caret: >= 1.2.3, < 2.0.0
>=1.2.3, <2.0  # Range expressions
1.2.3 || 1.4.5 # OR expressions
```

Resolved via **Masterminds/semver/v3**

### 12.3 Image Handling

Images are referenced in values and templates:
- No direct image pulling (delegated to K8s)
- Image pull policy configurable
- Pull secrets supported via values
- Registry authentication via K8s ImagePullSecrets

---

## 13. PLUGIN SYSTEM

### 13.1 Plugin Architecture

**File:** `/home/user/helmfire/analysis/sources/helm/internal/plugin/`

Plugins extend Helm via:
1. **Command Plugins** - Add new `helm` subcommands
2. **Post-Renderer Plugins** - Process rendered manifests
3. **Custom Getters** - Retrieve files from custom sources

### 13.2 Plugin Discovery

Plugins located in:
```
$HELM_PLUGINS (default: $HELM_DATA_HOME/plugins)
```

Plugin structure:
```
my-plugin/
├── plugin.yaml           # Metadata
├── bin/
│   └── helm-<name>      # Executable
└── <plugin-files>
```

### 13.3 Plugin Runtime

**File:** `/home/user/helmfire/analysis/sources/helm/internal/plugin/`

Runtimes:
- Shell scripts
- Binary executables
- WASM plugins (via Extism/WAZERO)

Communication:
- Environment variables for configuration
- stdin/stdout for data
- Exit codes for errors

---

## 14. TESTING AND FIXTURES

### 14.1 Test Utilities

**File:** `/home/user/helmfire/analysis/sources/helm/internal/test/`

Provides:
- Fake Kubernetes client (`pkg/kube/fake`)
- Memory driver for testing
- Test chart fixtures in `testdata/`
- Mock repositories

### 14.2 Test Fixtures

Chart examples in:
```
pkg/chart/v2/loader/testdata/
├── frobnitz_with_bom/      # Chart with UTF-8 BOM
├── frobnitz_with_symlink/  # Chart with symlinks
├── frobnitz_backslash/     # Windows path testing
└── [other variants]
```

---

## 15. ARCHITECTURAL PATTERNS

### 15.1 Dependency Injection

Configuration object passed to all actions:
```go
cfg := &action.Configuration{ ... }
install := action.NewInstall(cfg)
upgrade := action.NewUpgrade(cfg)
rollback := action.NewRollback(cfg)
```

Enables:
- Testability (swap components)
- Reusability (same config for multiple operations)
- Consistency (shared logger, K8s client)

### 15.2 Interfaces for Abstraction

Multiple chart versions hidden behind interfaces:
- `charter.Charter` - Any chart
- `chart.Accessor` - Chart operations
- `release.Releaser` - Any release
- `release.Accessor` - Release operations

Enables:
- Version compatibility
- Future extensibility
- Polymorphic operations

### 15.3 Action Pattern

Each operation (install, upgrade, etc.) is a struct with:
- Configuration reference
- Operation-specific options
- `Run()` or `RunWithContext()` method

Enables:
- Fluent API (set options then run)
- Type safety
- Testability

### 15.4 Storage Abstraction

Driver interface allows pluggable backends:
- Kubernetes Secrets (production)
- ConfigMaps (simple)
- Memory (testing)
- SQL (external)

Can be extended for custom backends.

---

## 16. ERROR HANDLING AND RECOVERY

### 16.1 Release Status Tracking

Releases track operation state:
- `deployed` - Successful
- `failed` - Error occurred
- `pending-upgrade` - Upgrade in progress
- `pending-rollback` - Rollback in progress
- `pending-install` - Install in progress
- `uninstalling` - Uninstall in progress

### 16.2 Dry Run Validation

Three dry-run strategies:
1. **Client-side** - Validate locally without cluster
2. **Server-side** - Server performs dry-run (safer)
3. **None** - Actual operation

Enables safe pre-deployment validation.

### 16.3 Cleanup on Failure

Option in Upgrade/Install:
```go
install.CleanupOnFail = true  // Delete resources if deployment fails
```

### 16.4 MaxHistory

Limit release revisions:
```go
cfg.Releases.MaxHistory = 10  // Keep last 10 revisions
```

Automatic cleanup of old revisions on Create.

---

## 17. PERFORMANCE CONSIDERATIONS

### 17.1 Caching

- Discovery client caching (K8s API types)
- Repository index caching
- Chart metadata caching

### 17.2 Parallel Operations

- Concurrent chart downloads
- Parallel resource patching
- Concurrent status checking (with configured burst limits)

**Configuration:**
```bash
HELM_BURST_LIMIT    # Default 100
HELM_QPS            # Queries per second
```

### 17.3 Resource Efficiency

- Lazy initialization of clients
- Memory driver for testing (no persistence overhead)
- Streaming YAML parsing
- Delta-based updates (strategic merge patch)

---

## SUMMARY TABLE: REUSABLE COMPONENTS

| Component | Package | Key Types | Use Cases |
|-----------|---------|-----------|-----------|
| Chart Loading | `pkg/chart/v2/loader` | `Chart`, `Loader` | Parse charts, load from disk/archives |
| Rendering | `pkg/engine` | `Engine` | Render templates with Sprig functions |
| Values | `pkg/chart/common/util`, `pkg/cli/values` | `Values`, `Options` | Merge values, coalesce defaults |
| K8s Client | `pkg/kube` | `Client`, `Interface` | Apply resources, wait for readiness |
| Storage | `pkg/storage` | `Storage`, `Driver` | Store/retrieve release history |
| Actions | `pkg/action` | `Install`, `Upgrade`, `Rollback` | Execute release operations |
| Repositories | `pkg/repo/v1` | `ChartRepo`, `Index` | Manage chart repositories |
| Registry | `pkg/registry` | `Client` | Push/pull OCI charts |
| Downloader | `pkg/downloader` | `Manager` | Resolve and download dependencies |
| Post Render | `pkg/postrenderer` | `PostRenderer` | Modify rendered manifests |

---

## CONCLUSION

The Helm project is a well-architected package manager with clear separation of concerns:

1. **Chart Management** - Abstraction layer supporting multiple versions
2. **Template Engine** - Powerful rendering with Sprig and K8s functions
3. **Kubernetes Integration** - Full-featured client with SSA support
4. **Release Management** - Rich action framework with hooks and waiting
5. **Storage** - Pluggable backends from K8s Secrets to SQL
6. **Extensibility** - Plugins and post-renderers for customization

For **helmfire**, the key reusable components are:
- Chart loading and rendering
- Release action framework
- Kubernetes client integration
- Storage backends
- Repository and registry handling

These can be imported as libraries without needing the CLI infrastructure.


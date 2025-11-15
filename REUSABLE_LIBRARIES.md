# Reusable Libraries Reference for Helmfire

This document consolidates all reusable components from Helm and Helmfile that can be imported as libraries for the Helmfire project.

---

## Table of Contents

1. [Helmfile Libraries](#helmfile-libraries)
2. [Helm Libraries](#helm-libraries)
3. [Integration Examples](#integration-examples)
4. [Dependency Map](#dependency-map)
5. [Version Compatibility](#version-compatibility)

---

## Helmfile Libraries

### 1. State Management (`pkg/state`)

**Import:** `github.com/helmfile/helmfile/pkg/state`

**Key Types:**
```go
type HelmState struct {
    FilePath       string
    Releases       []ReleaseSpec
    Repositories   []RepositorySpec
    Environments   map[string]EnvironmentSpec
    // ...
}

type ReleaseSpec struct {
    Name        string
    Namespace   string
    Chart       string
    Version     string
    Values      []any
    Needs       []string  // Dependencies
    // ...
}
```

**Key Functions:**
```go
// Create state from YAML
func NewCreator(...) *StateCreator
func (sc *StateCreator) Parse(content []byte, basedir, filename string) (*HelmState, error)

// Execute operations
func (st *HelmState) SyncReleases(...) []error
func (st *HelmState) DiffReleases(...) (bool, []error)
func (st *HelmState) PrepareCharts(...) (map[PrepareChartKey]string, error)
func (st *HelmState) PlanReleases(opts PlanOptions) ([][]Release, error)
```

**Use Cases for Helmfire:**
- Parse helmfile.yaml files
- Build release dependency DAG
- Execute sync operations
- Detect changes via diff

**File References:**
- Main: `analysis/sources/helmfile/pkg/state/state.go`
- Creation: `analysis/sources/helmfile/pkg/state/create.go`
- Execution: `analysis/sources/helmfile/pkg/state/state_run.go`

---

### 2. Application Layer (`pkg/app`)

**Import:** `github.com/helmfile/helmfile/pkg/app`

**Key Types:**
```go
type App struct {
    OverrideKubeContext string
    OverrideHelmBinary  string
    Logger              *zap.SugaredLogger
    Env                 string
    Namespace           string
    FileOrDir           string
    Selectors           []string
    // ...
}

type Run struct {
    state          *state.HelmState
    helm           helmexec.Interface
    ReleaseToChart map[state.PrepareChartKey]string
}
```

**Key Functions:**
```go
func New(conf ConfigProvider) *App
func (a *App) Sync(c SyncConfigProvider) error
func (a *App) Apply(c ApplyConfigProvider) error
func (a *App) Diff(c DiffConfigProvider) error
func (a *App) ForEachState(do func(*Run) (bool, []error), ...) error
```

**Use Cases for Helmfire:**
- High-level orchestration
- State file loading with overrides
- Multi-environment support
- Selector-based filtering

**File References:**
- Main: `analysis/sources/helmfile/pkg/app/app.go`
- Run: `analysis/sources/helmfile/pkg/app/run.go`

---

### 3. Helm Execution (`pkg/helmexec`)

**Import:** `github.com/helmfile/helmfile/pkg/helmexec`

**Key Types:**
```go
type Interface interface {
    AddRepo(ctx HelmContext, name, url, username, password string, ...) error
    UpdateRepo(ctx HelmContext) error
    Diff(ctx HelmContext, release, chart string, ...) (bool, error)
    Template(ctx HelmContext, release, chart string, ...) ([]byte, error)
    Chart(ctx HelmContext, chart, version string) (*chart.Chart, error)
    // ... many more
}

type Runner interface {
    Execute(cmd string, args []string, env map[string]string, showOutput bool) ([]byte, error)
}
```

**Key Functions:**
```go
func New(helmBinary string, opts HelmExecOptions, ...) (*execer, error)
func GetHelmVersion(helmBinary string, runner Runner) (*semver.Version, error)
```

**Use Cases for Helmfire:**
- Execute helm commands
- Version detection
- Command output capture

**File References:**
- Main: `analysis/sources/helmfile/pkg/helmexec/exec.go`
- Runner: `analysis/sources/helmfile/pkg/helmexec/runner.go`

---

### 4. Event System (`pkg/event`)

**Import:** `github.com/helmfile/helmfile/pkg/event`

**Key Types:**
```go
type Hook struct {
    Name     string
    Events   []string  // pre-sync, post-sync, etc.
    Command  string
    Args     []string
    ShowLogs bool
}

type Bus struct {
    Runner  helmexec.Runner
    Hooks   []Hook
    Logger  *zap.SugaredLogger
}
```

**Key Functions:**
```go
func (bus *Bus) Trigger(evt string, evtErr error, context map[string]any) (bool, error)
```

**Available Events:**
- `presync`, `postsync`
- `preapply`, `postapply`
- `prediff`, `postdiff`
- `presyncrelease`, `postsyncrelease`

**Use Cases for Helmfire:**
- Execute lifecycle hooks
- Custom actions on events
- Integration with external tools

**File References:**
- Main: `analysis/sources/helmfile/pkg/event/bus.go`

---

### 5. Environment Management (`pkg/environment`)

**Import:** `github.com/helmfile/helmfile/pkg/environment`

**Key Types:**
```go
type Environment struct {
    Name     string
    Values   map[string]any
    Defaults map[string]any
}
```

**Use Cases for Helmfire:**
- Environment-specific values
- Value lookup and merging

**File References:**
- Main: `analysis/sources/helmfile/pkg/environment/`

---

### 6. YAML Utilities (`pkg/yaml`)

**Import:** `github.com/helmfile/helmfile/pkg/yaml`

**Key Functions:**
```go
func NewDecoder(data []byte, strict bool) func(any) error
func Unmarshal(data []byte, v any) error
func Marshal(v any) ([]byte, error)
```

**Use Cases for Helmfire:**
- YAML parsing with strict mode
- Compatible with helmfile's YAML handling

**File References:**
- Main: `analysis/sources/helmfile/pkg/yaml/yaml.go`

---

## Helm Libraries

### 1. Chart Loading (`pkg/chart/v2/loader`)

**Import:** `helm.sh/helm/v4/pkg/chart/v2/loader`

**Key Functions:**
```go
func Load(name string) (*chart.Chart, error)
func LoadFiles(files []*archive.BufferedFile) (*chart.Chart, error)
func LoadDir(dir string) (*chart.Chart, error)
```

**Chart Structure:**
```go
type Chart struct {
    Metadata     *Metadata
    Templates    []*common.File
    Values       map[string]interface{}
    Schema       []byte
    Files        []*common.File
    dependencies []*Chart
}
```

**Use Cases for Helmfire:**
- Load local charts for substitution
- Parse chart metadata
- Validate chart structure

**File References:**
- Main: `analysis/sources/helm/pkg/chart/v2/loader/load.go`
- Chart: `analysis/sources/helm/pkg/chart/v2/chart.go`

---

### 2. Template Rendering (`pkg/engine`)

**Import:** `helm.sh/helm/v4/pkg/engine`

**Key Types:**
```go
type Engine struct {
    Strict              bool
    LintMode            bool
    EnableDNS           bool
    CustomTemplateFuncs template.FuncMap
}
```

**Key Functions:**
```go
func (e Engine) Render(chrt Charter, values Values) (map[string]string, error)
func New(cfg RESTClientGetter) *Engine
```

**Use Cases for Helmfire:**
- Render templates locally
- Preview changes before applying
- Test substitutions

**File References:**
- Main: `analysis/sources/helm/pkg/engine/engine.go`
- Functions: `analysis/sources/helm/pkg/engine/funcs.go`

---

### 3. Values Handling (`pkg/cli/values`, `pkg/chart/common/util`)

**Import:**
- `helm.sh/helm/v4/pkg/cli/values`
- `helm.sh/helm/v4/pkg/chart/common/util`

**Key Types:**
```go
type Options struct {
    ValueFiles    []string
    StringValues  []string
    Values        []string
    FileValues    []string
    JSONValues    []string
}
```

**Key Functions:**
```go
func (opts *Options) MergeValues(p getter.Providers) (map[string]interface{}, error)
func CoalesceValues(chrt Charter, vals map[string]interface{}) (Values, error)
func ToRenderValues(chrt Charter, values Values, options ReleaseOptions, caps *Capabilities) (Values, error)
```

**Use Cases for Helmfire:**
- Merge multiple values files
- Combine with --set overrides
- Validate against schema

**File References:**
- Options: `analysis/sources/helm/pkg/cli/values/options.go`
- Coalesce: `analysis/sources/helm/pkg/chart/common/util/coalesce.go`

---

### 4. Kubernetes Client (`pkg/kube`)

**Import:** `helm.sh/helm/v4/pkg/kube`

**Key Types:**
```go
type Interface interface {
    Create(resources ResourceList, options ...ClientCreateOption) (*Result, error)
    Update(original, target ResourceList, options ...ClientUpdateOption) (*Result, error)
    Delete(resources ResourceList, policy metav1.DeletionPropagation) (*Result, []error)
    Build(reader io.Reader, validate bool) (ResourceList, error)
    GetWaiter(ws WaitStrategy) (Waiter, error)
}
```

**Key Functions:**
```go
func New(getter genericclioptions.RESTClientGetter) *Client
func (c *Client) Create(resources ResourceList, ...) (*Result, error)
func (c *Client) Update(original, target ResourceList, ...) (*Result, error)
```

**Use Cases for Helmfire:**
- Apply rendered manifests
- Wait for resource readiness
- Server-side apply support

**File References:**
- Interface: `analysis/sources/helm/pkg/kube/interface.go`
- Client: `analysis/sources/helm/pkg/kube/client.go`
- Wait: `analysis/sources/helm/pkg/kube/wait.go`

---

### 5. Release Storage (`pkg/storage`)

**Import:** `helm.sh/helm/v4/pkg/storage`

**Key Types:**
```go
type Storage struct {
    driver.Driver
    MaxHistory int
}
```

**Key Functions:**
```go
func Init(d driver.Driver) *Storage
func (s *Storage) Get(name string, version int) (release.Releaser, error)
func (s *Storage) Create(rls release.Releaser) error
func (s *Storage) Update(rls release.Releaser) error
func (s *Storage) ListReleases() ([]release.Releaser, error)
```

**Available Drivers:**
- Secrets (default): `pkg/storage/driver/secrets.go`
- ConfigMaps: `pkg/storage/driver/cfgmaps.go`
- Memory: `pkg/storage/driver/memory.go`
- SQL: `pkg/storage/driver/sql.go`

**Use Cases for Helmfire:**
- Query release history
- Track deployment state
- Implement drift detection

**File References:**
- Storage: `analysis/sources/helm/pkg/storage/storage.go`
- Drivers: `analysis/sources/helm/pkg/storage/driver/`

---

### 6. Action Framework (`pkg/action`)

**Import:** `helm.sh/helm/v4/pkg/action`

**Key Types:**
```go
type Configuration struct {
    RESTClientGetter    RESTClientGetter
    Releases            *storage.Storage
    KubeClient          kube.Interface
    RegistryClient      *registry.Client
    Capabilities        *common.Capabilities
}

type Install struct {
    cfg             *Configuration
    Namespace       string
    Wait            bool
    Timeout         time.Duration
    ServerSideApply bool
}

type Upgrade struct { /* similar */ }
type Rollback struct { /* similar */ }
```

**Key Functions:**
```go
func NewConfiguration() *Configuration
func (c *Configuration) Init(getter genericclioptions.RESTClientGetter, namespace, driver string) error

func NewInstall(cfg *Configuration) *Install
func (i *Install) Run(chrt *chart.Chart, vals map[string]interface{}) (*release.Release, error)

func NewUpgrade(cfg *Configuration) *Upgrade
func (u *Upgrade) Run(name string, chrt *chart.Chart, vals map[string]interface{}) (*release.Release, error)

func NewStatus(cfg *Configuration) *Status
func (s *Status) Run(name string) (*release.Release, error)
```

**Use Cases for Helmfire:**
- Install/upgrade releases
- Query release status
- Implement rollback

**File References:**
- Config: `analysis/sources/helm/pkg/action/action.go`
- Install: `analysis/sources/helm/pkg/action/install.go`
- Upgrade: `analysis/sources/helm/pkg/action/upgrade.go`
- Status: `analysis/sources/helm/pkg/action/status.go`

---

### 7. Post-Renderer (`pkg/postrenderer`)

**Import:** `helm.sh/helm/v4/pkg/postrenderer`

**Key Types:**
```go
type PostRenderer interface {
    Run(renderedManifests *bytes.Buffer) (modifiedManifests *bytes.Buffer, err error)
}
```

**Use Cases for Helmfire:**
- Implement image substitution
- Modify manifests after rendering
- Add annotations/labels

**Implementation Example:**
```go
type ImageSubstitutor struct {
    replacements map[string]string
}

func (is *ImageSubstitutor) Run(manifests *bytes.Buffer) (*bytes.Buffer, error) {
    // Parse YAML, replace images, return modified YAML
}
```

**File References:**
- Interface: `analysis/sources/helm/pkg/postrenderer/postrenderer.go`

---

### 8. Registry Client (`pkg/registry`)

**Import:** `helm.sh/helm/v4/pkg/registry`

**Key Types:**
```go
type Client struct {
    credentialsFile string
    enableCache     bool
}
```

**Key Functions:**
```go
func NewClient(opts ...ClientOption) (*Client, error)
func (c *Client) Login(ctx context.Context, hostname string, ...) error
func (c *Client) Pull(ctx context.Context, ref string, ...) (*chart.Chart, error)
func (c *Client) Push(ctx context.Context, ref string, chart *chart.Chart, ...) (digest string, err error)
```

**Use Cases for Helmfire:**
- Pull charts from OCI registries
- Push modified charts
- Registry authentication

**File References:**
- Main: `analysis/sources/helm/pkg/registry/client.go`

---

### 9. Repository Management (`pkg/repo/v1`)

**Import:** `helm.sh/helm/v4/pkg/repo/v1`

**Key Types:**
```go
type ChartRepository struct {
    Config *config.RepositoryEntry
    Client *http.Client
}

type IndexFile struct {
    APIVersion string
    Entries    map[string]ChartVersions
}
```

**Key Functions:**
```go
func (r *ChartRepository) DownloadIndexFile(cachePath string) error
func (r *ChartRepository) Index() (*IndexFile, error)
func (i *IndexFile) Get(name, version string) (*ChartVersion, error)
```

**Use Cases for Helmfire:**
- Resolve chart references
- Download chart metadata
- Cache repository indexes

**File References:**
- Main: `analysis/sources/helm/pkg/repo/v1/`

---

## Integration Examples

### Example 1: Load and Parse Helmfile

```go
package main

import (
    "github.com/helmfile/helmfile/pkg/state"
    "github.com/helmfile/helmfile/pkg/app"
    "go.uber.org/zap"
)

func loadHelmfile(path string) (*state.HelmState, error) {
    // Create logger
    logger, _ := zap.NewDevelopment()
    sugar := logger.Sugar()

    // Create state creator
    creator := state.NewCreator(
        sugar,
        filesystem.DefaultFileSystem,
        valsRuntime,
        getHelm,
        "helm",
        "",
        nil,
        false,
        "",
    )

    // Read file
    content, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    // Parse
    helmState, err := creator.Parse(content, filepath.Dir(path), path)
    if err != nil {
        return nil, err
    }

    return helmState, nil
}
```

### Example 2: Execute Sync with Substitutions

```go
func syncWithSubstitution(helmState *state.HelmState, chartSub map[string]string) error {
    // Apply chart substitutions
    for i := range helmState.Releases {
        if localPath, ok := chartSub[helmState.Releases[i].Chart]; ok {
            helmState.Releases[i].Chart = localPath
        }
    }

    // Sync releases
    errs := helmState.SyncReleases(
        &helmexec.ShellRunner{},
        helmState.Releases,
        0, // concurrency
    )

    if len(errs) > 0 {
        return fmt.Errorf("sync failed: %v", errs)
    }

    return nil
}
```

### Example 3: Image Substitution Post-Renderer

```go
type ImageSubstitutor struct {
    replacements map[string]string
}

func (is *ImageSubstitutor) Run(manifests *bytes.Buffer) (*bytes.Buffer, error) {
    // Split into individual YAML documents
    docs := strings.Split(manifests.String(), "---")
    var modified []string

    for _, doc := range docs {
        if strings.TrimSpace(doc) == "" {
            continue
        }

        // Parse YAML
        var obj map[string]interface{}
        if err := yaml.Unmarshal([]byte(doc), &obj); err != nil {
            return nil, err
        }

        // Replace images in containers
        is.replaceImages(obj)

        // Marshal back
        modifiedYAML, err := yaml.Marshal(obj)
        if err != nil {
            return nil, err
        }
        modified = append(modified, string(modifiedYAML))
    }

    return bytes.NewBufferString(strings.Join(modified, "---\n")), nil
}

func (is *ImageSubstitutor) replaceImages(obj map[string]interface{}) {
    // Navigate to spec.template.spec.containers[].image
    if spec, ok := obj["spec"].(map[string]interface{}); ok {
        if template, ok := spec["template"].(map[string]interface{}); ok {
            if podSpec, ok := template["spec"].(map[string]interface{}); ok {
                if containers, ok := podSpec["containers"].([]interface{}); ok {
                    for i := range containers {
                        if container, ok := containers[i].(map[string]interface{}); ok {
                            if image, ok := container["image"].(string); ok {
                                if replacement, exists := is.replacements[image]; exists {
                                    container["image"] = replacement
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
```

### Example 4: Drift Detection

```go
func detectDrift(helmState *state.HelmState, helm helmexec.Interface) ([]DriftReport, error) {
    var reports []DriftReport

    // Run diff for all releases
    hasDiff, errs := helmState.DiffReleases(
        helm,
        helmState.Releases,
        0, // concurrency
        false, // suppress output
        false, // suppress secrets
        false, // show secrets
        false, // suppress diff
        false, // context
    )

    if len(errs) > 0 {
        return nil, fmt.Errorf("diff failed: %v", errs)
    }

    if hasDiff {
        // Parse diff output and create reports
        for _, release := range helmState.Releases {
            report := DriftReport{
                Timestamp:   time.Now(),
                ReleaseName: release.Name,
                Namespace:   release.Namespace,
                DriftType:   DriftTypeConfiguration,
                Severity:    SeverityMedium,
            }
            reports = append(reports, report)
        }
    }

    return reports, nil
}
```

### Example 5: File Watching

```go
import "github.com/fsnotify/fsnotify"

func watchHelmfile(path string, onChange func()) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }
    defer watcher.Close()

    if err := watcher.Add(path); err != nil {
        return err
    }

    debounce := time.NewTimer(500 * time.Millisecond)
    debounce.Stop()

    for {
        select {
        case event := <-watcher.Events:
            if event.Op&fsnotify.Write == fsnotify.Write {
                debounce.Reset(500 * time.Millisecond)
            }

        case <-debounce.C:
            onChange()

        case err := <-watcher.Errors:
            return err
        }
    }
}
```

---

## Dependency Map

### Helmfire Dependencies

```
helmfire
├── github.com/helmfile/helmfile
│   ├── pkg/state         (required)
│   ├── pkg/app           (required)
│   ├── pkg/helmexec      (required)
│   ├── pkg/event         (optional)
│   └── pkg/environment   (optional)
│
├── helm.sh/helm/v4
│   ├── pkg/chart/v2/loader    (required)
│   ├── pkg/engine             (optional)
│   ├── pkg/cli/values         (required)
│   ├── pkg/chart/common/util  (required)
│   ├── pkg/kube               (optional)
│   ├── pkg/storage            (optional)
│   ├── pkg/action             (optional)
│   └── pkg/postrenderer       (required)
│
├── github.com/fsnotify/fsnotify (required)
├── github.com/spf13/cobra       (required)
├── go.uber.org/zap              (required)
└── gopkg.in/yaml.v3             (required)
```

### Transitive Dependencies

Both Helm and Helmfile bring in:
- `k8s.io/client-go` - Kubernetes client
- `k8s.io/apimachinery` - Kubernetes types
- `github.com/Masterminds/sprig/v3` - Template functions
- `github.com/Masterminds/semver/v3` - Version handling
- `sigs.k8s.io/yaml` - YAML processing

---

## Version Compatibility

### Helmfile

**Current Version:** v0.x (varies)
**Stability:** Stable API for core packages
**Breaking Changes:** Rare in pkg/state, pkg/app

**Recommendation:**
- Pin to specific version initially
- Test upgrades in isolation
- Monitor helmfile releases

### Helm

**Current Version:** v4.x (based on analysis)
**Stability:** Stable API for pkg/ packages
**Breaking Changes:** Versioned packages (v2, v3, v4)

**Recommendation:**
- Use v4 packages where available
- Fall back to v3 for compatibility
- Pin to minor version

### Go Version

**Minimum:** Go 1.21 (for generics, updated packages)
**Recommended:** Go 1.24+

---

## Best Practices

### 1. Import Only What You Need

```go
// Good
import "github.com/helmfile/helmfile/pkg/state"

// Avoid
import "github.com/helmfile/helmfile/pkg/app"  // if not using App directly
```

### 2. Vendor Dependencies

```bash
go mod vendor
```

Ensures reproducible builds and insulation from upstream changes.

### 3. Wrap External APIs

```go
// Create internal interfaces that wrap external types
type StateManager interface {
    Load() error
    Sync() error
}

type helmfileStateManager struct {
    state *state.HelmState
}

func (h *helmfileStateManager) Load() error {
    // Wrap helmfile state loading
}
```

### 4. Test with Mocks

Both Helm and Helmfile provide interfaces that can be mocked:
- `helmexec.Interface` - Mock helm execution
- `kube.Interface` - Mock Kubernetes client
- `storage.Driver` - Mock storage

### 5. Handle Breaking Changes

```go
// Version guard
if helmfileVersion.Compare("v0.150.0") >= 0 {
    // Use new API
} else {
    // Use old API
}
```

---

## Summary Table

| Library | Priority | Purpose | Package |
|---------|----------|---------|---------|
| Helmfile State | **High** | Parse helmfile.yaml, execute sync | `github.com/helmfile/helmfile/pkg/state` |
| Helmfile App | **High** | Orchestration layer | `github.com/helmfile/helmfile/pkg/app` |
| Helmfile HelmExec | **High** | Helm command execution | `github.com/helmfile/helmfile/pkg/helmexec` |
| Helm Chart Loader | **High** | Load local charts | `helm.sh/helm/v4/pkg/chart/v2/loader` |
| Helm Values | **High** | Merge values | `helm.sh/helm/v4/pkg/cli/values` |
| Helm Post-Renderer | **High** | Image substitution | `helm.sh/helm/v4/pkg/postrenderer` |
| Helm Engine | Medium | Template rendering | `helm.sh/helm/v4/pkg/engine` |
| Helm Kube | Medium | Kubernetes operations | `helm.sh/helm/v4/pkg/kube` |
| Helm Storage | Medium | Release tracking | `helm.sh/helm/v4/pkg/storage` |
| Helm Action | Low | Direct helm operations | `helm.sh/helm/v4/pkg/action` |
| Helmfile Event | Low | Lifecycle hooks | `github.com/helmfile/helmfile/pkg/event` |

---

**Document Version:** 1.0
**Last Updated:** 2025-11-15
**Next Review:** When upgrading major dependencies

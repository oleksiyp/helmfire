# Helmfile Project Analysis

**Project Location:** `/home/user/helmfire/analysis/sources/helmfile`  
**Language:** Go 1.24.6  
**Total Go Files:** 169  
**Architecture:** Modular with clear separation between CLI commands, application logic, and state management

## Table of Contents

1. [Project Structure](#project-structure)
2. [Core Components](#core-components)
3. [Key Files and Entry Points](#key-files-and-entry-points)
4. [Data Structures](#data-structures)
5. [Dependencies](#dependencies)
6. [Reusable Components for Helmfire](#reusable-components-for-helmfire)

---

## Project Structure

### Top-Level Organization

```
helmfile/
├── main.go                    # Entry point with signal handling
├── cmd/                       # Command implementations (18 files)
├── pkg/                       # Core packages (23 directories)
├── examples/                  # Example helmfile configurations
├── docs/                      # Documentation
├── go.mod / go.sum           # Dependencies
└── Makefile                  # Build configuration
```

### Package Organization

```
pkg/
├── app/                  # Application logic and orchestration (35 files, ~15,915 lines)
│   ├── app.go           # Main App struct and high-level operations
│   ├── run.go           # Release execution runner
│   ├── context.go       # Shared context (repo sync tracking)
│   ├── desired_state_file_loader.go  # YAML file loading
│   ├── load_opts.go     # Loading options
│   ├── formatters.go    # Output formatting
│   └── {command}_test.go # Command implementation tests
│
├── state/               # State management and helm operations (29 files, ~4,369 lines core)
│   ├── state.go         # HelmState struct and core operations (4,369 lines)
│   ├── state_run.go     # Execution logic (concurrency, DAG, scatter-gather)
│   ├── release.go       # ReleaseSpec definitions and operations
│   ├── create.go        # StateCreator for YAML parsing (479 lines)
│   ├── environment.go   # Environment specification
│   ├── types.go         # Template data types
│   ├── helmx.go         # Helm execution helpers
│   ├── chart_dependency.go  # Chart dependency management
│   └── {operation}_test.go  # Comprehensive tests
│
├── helmexec/            # Helm binary execution (12 files)
│   ├── exec.go          # HelmExecOptions, command building (23,697 lines)
│   ├── runner.go        # Command execution interface
│   ├── helmexec.go      # Interface definitions
│   └── exit_error.go    # Error handling
│
├── config/              # Configuration management
│   └── config.go        # CLI configuration parsing
│
├── event/               # Event/Hook system
│   └── bus.go           # Hook execution engine
│
├── environment/         # Environment variable management
│   └── *.go             # Environment value handling
│
├── tmpl/                # Template rendering
│   └── *.go             # Go template execution
│
├── yaml/                # YAML encoding/decoding
│   └── yaml.go          # YAML marshaling/unmarshaling
│
├── remote/              # Remote file handling
│   └── *.go             # Remote resource management
│
├── filesystem/          # File system abstraction
│   └── *.go             # File operations
│
└── [other packages]     # plugins, policy, hcllang, etc.
```

---

## Core Components

### 1. Helmfile Sync Mechanism

**How Sync Works Internally:**

The sync operation is the primary command for deploying releases. Here's the internal flow:

#### Entry Point: `/home/user/helmfire/analysis/sources/helmfile/cmd/sync.go` (Lines 10-58)
```go
func NewSyncCmd(globalCfg *config.GlobalImpl) *cobra.Command {
    // Parses sync-specific flags
    // Creates SyncImpl from configuration
    // Calls app.Sync(syncImpl)
}
```

#### Main Application Sync: `/home/user/helmfire/analysis/sources/helmfile/pkg/app/app.go` (Lines 358-383)
```go
func (a *App) Sync(c SyncConfigProvider) error {
    return a.ForEachState(func(run *Run) (ok bool, errs []error) {
        // Prepares charts with options
        // Calls a.sync(run, c)
    }, c.IncludeTransitiveNeeds())
}
```

**Key Sync Flow:**

1. **State Loading** → Load helmfile.yaml files with template rendering
2. **Release Selection** → Apply selectors and filter releases
3. **Chart Preparation** → Download/process charts with dependencies
4. **DAG Building** → Create dependency graph from "needs" declarations
5. **Batch Execution** → Execute releases in topologically sorted batches
6. **Helm Commands** → Execute `helm upgrade --install` for each release

**Core State Management:** `/home/user/helmfire/analysis/sources/helmfile/pkg/state/state.go`

Major methods include:
- Line 591: `SyncRepos()` - Update helm repositories
- Line 647: `prepareSyncReleases()` - Prepare releases for sync
- Line 752: `isReleaseInstalled()` - Check release status
- Line 975: `SyncReleases()` - Main sync execution
- Line 1109: `performSyncOrReinstallOfRelease()` - Individual release sync

**Concurrency Model:** `/home/user/helmfire/analysis/sources/helmfile/pkg/state/state_run.go` (Lines 20-90)

Uses scatter-gather pattern for parallel execution:
- Line 20: `scatterGather()` - Generic concurrency pattern
- Line 44: `scatterGatherReleases()` - Release-specific concurrency
- Line 50: `iterateOnReleases()` - Worker pool pattern
- Configurable concurrency via `--concurrency` flag

### 2. File Watching and Monitoring

**Finding:** Helmfile does NOT have built-in file watching. There is no watch command or file monitoring mechanism in the codebase.

**Verification:**
```bash
grep -r "watch\|Watch" /home/user/helmfire/analysis/sources/helmfile --include="*.go"
# Returns no results
```

**Implication for Helmfire:** This is where helmfire can add value by implementing:
- File system monitoring (helmfile.yaml, values files, subhelmfiles)
- Trigger automatic sync on changes
- Manifest drift detection

### 3. YAML Parsing and Processing

**YAML Decoder:** `/home/user/helmfire/analysis/sources/helmfile/pkg/yaml/yaml.go`

Supports both YAML v2 and v3 with configurable strict mode:
```go
func NewDecoder(data []byte, strict bool) func(any) error
func Unmarshal(data []byte, v any) error
func Marshal(v any) ([]byte, error)
```

**State Creation & Parsing:** `/home/user/helmfire/analysis/sources/helmfile/pkg/state/create.go` (Lines 56-97)

`StateCreator` struct:
- Parses YAML content into HelmState
- Handles file includes and inheritance
- Supports HCL and Go template syntax
- Resolves environment variables via `vals` library
- Strict mode validation

**Template Rendering:** `/home/user/helmfire/analysis/sources/helmfile/pkg/state/state_exec_tmpl.go`

- Executes Go text/template expressions
- Available variables:
  - `.Environment` - Environment variables and values
  - `.Namespace` - Release namespace
  - `.Values` - Helmfile-wide state values
  - `.Release` - Release-specific data

**Flow for Processing helmfile.yaml:**

1. File is read and processed by `desiredStateLoader.Load()`
2. Content is passed to `StateCreator.Parse()`
3. YAML is unmarshaled into `HelmState` struct
4. Template expressions are executed
5. Subhelmfiles (via `helmfiles:` key) are recursively loaded
6. Releases are filtered by selectors and conditions
7. Release-level templates and values are rendered

### 4. Helm Integration Points

**Helm Execution:** `/home/user/helmfire/analysis/sources/helmfile/pkg/helmexec/exec.go` (Lines 1-150)

Main entry:
```go
func New(helmBinary string, options HelmExecOptions, logger *zap.SugaredLogger, 
         kubeconfig string, kubeContext string, runner Runner) (*execer, error)
```

**Key Helm Commands Built by Helmfile:**

From state.go:
- `helm repo add/update` (Line 591: `SyncRepos()`)
- `helm dependency build` (Chart preparation)
- `helm upgrade --install` (Line 975: `SyncReleases()`)
- `helm delete` (Line 885: `DeleteReleasesForSync()`)
- `helm diff` (Line 2217: `DiffReleases()`)
- `helm template` (Line 1663: `TemplateReleases()`)
- `helm list` (Line 1180: `listReleases()`)
- `helm status` (Line 2316: `ReleaseStatuses()`)

**Helm Version Detection:** `/home/user/helmfire/analysis/sources/helmfile/pkg/helmexec/exec.go` (Lines 95-103)

```go
func GetHelmVersion(helmBinary string, runner Runner) (*semver.Version, error)
// Uses: helm version --client --short
// Parses semantic version for feature compatibility
```

---

## Key Files and Entry Points

### Main Entry Point: `/home/user/helmfire/analysis/sources/helmfile/main.go` (50 lines)

```go
func main() {
    // Signal handling (SIGINT, SIGTERM)
    // Creates root command
    // Executes with global context cancellation
    // Handles exit codes (130 for SIGINT, 143 for SIGTERM)
}
```

Global variables:
- `app.Cancel` - Context cancellation function
- `app.CleanWaitGroup` - Cleanup synchronization

### Root Command: `/home/user/helmfire/analysis/sources/helmfile/cmd/root.go` (147 lines)

**Function:** `NewRootCmd(globalConfig *config.GlobalOptions)`

**Registered Subcommands:**
```
apply      - Apply releases with change detection
build      - Build release chart
cache      - Manage helm chart cache
deps       - Update chart dependencies
destroy    - Delete releases
diff       - Show manifest differences
fetch      - Fetch charts to local directory
init       - Initialize helmfile config
lint       - Validate release manifests
list       - List all releases
repos      - List/manage repositories
status     - Show release status
sync       - Sync releases (upgrade/install)
template   - Render release templates
test       - Test releases
write-values - Write computed values
show-dag   - Display dependency graph
version    - Show version
```

**Global Flags:**
```
--helm-binary (-b)             # Path to helm executable
--kustomize-binary (-k)        # Path to kustomize executable
--file (-f)                    # helmfile.yaml path or directory
--environment (-e)             # Environment name
--namespace (-n)               # Override namespace
--chart (-c)                   # Override chart
--selector (-l)                # Release selector filter
--kubeconfig                   # Kubernetes config path
--kube-context                 # Kubernetes context
--state-values-set             # Override helmfile values
--state-values-file            # Values file
--debug                        # Debug output
--color                        # Colored output
--quiet (-q)                   # Suppress output
--log-level                    # Log level (debug/info/warn/error)
--interactive (-i)             # Require confirmation
```

### App Initialization: `/home/user/helmfire/analysis/sources/helmfile/pkg/app/app.go` (Lines 72-110)

```go
func New(conf ConfigProvider) *App {
    // Creates App struct with configuration
    // Initializes vals runtime for value rendering
    // Sets up helm instances cache
}
```

**App Struct Fields:**
```go
type App struct {
    OverrideKubeContext, OverrideHelmBinary, OverrideKustomizeBinary string
    EnableLiveOutput, StripArgsValuesOnExitError, DisableForceUpdate bool
    Logger *zap.SugaredLogger
    Kubeconfig, Env, Namespace, Chart, Args string
    Selectors []string
    ValuesFiles []string
    Set map[string]any
    FileOrDir string
    helms map[helmKey]helmexec.Interface  // Helm cache per kubecontext
    ctx goContext.Context                  // Cancellable context
}
```

### State Loading & Orchestration: `/home/user/helmfire/analysis/sources/helmfile/pkg/app/app.go` (Lines 1127-1143)

```go
func (a *App) ForEachState(do func(*Run) (bool, []error), 
                          includeTransitiveNeeds bool, 
                          o ...LoadOption) error {
    // Visits each state file (helmfile.yaml)
    // Loads releases with selectors
    // Executes callback for each state
}
```

**Loading Pipeline:**
1. `visitStatesWithSelectorsAndRemoteSupportWithContext()` (Line 1244)
   - Resolves file/directory patterns
   - Loads state files with dependency tracking
   - Handles remote helmfiles (via go-getter)
   - Applies selectors to filter releases

2. `loadFileWithOverrides()` in desiredStateLoader
   - Processes environment values
   - Renders templates
   - Loads subhelmfiles recursively
   - Applies overrides (namespace, kube-context, etc.)

### Run Execution: `/home/user/helmfire/analysis/sources/helmfile/pkg/app/run.go` (Lines 16-58)

```go
type Run struct {
    state *state.HelmState
    helm helmexec.Interface
    ctx *Context
    ReleaseToChart map[state.PrepareChartKey]string
    Ask func(string) bool  // For interactive mode
}

func (r *Run) withPreparedCharts(helmfileCommand string, opts state.ChartPrepareOptions, f func()) error {
    // Syncs repositories
    // Prepares charts (downloads, processes)
    // Executes callback with prepared charts
}
```

### Release Management: `/home/user/helmfire/analysis/sources/helmfile/pkg/state/release.go`

**ReleaseSpec Struct:** `/home/user/helmfire/analysis/sources/helmfile/pkg/state/state.go` (Lines 247-431)

Major fields:
```go
type ReleaseSpec struct {
    Name, Namespace, Chart, Version string
    Installed *bool                  // Inclusion condition
    Condition string                 // Boolean template expression
    Labels map[string]string         // Release labels
    Values []any                     // YAML/go-getter values
    Secrets []any                    // Secret values
    SetValues []SetValue             // --set style values
    SetStringValues []SetValue       // --set-string style values
    Needs []string                   // Dependency declarations
    Hooks []event.Hook               // Lifecycle hooks
    
    // Advanced features
    Dependencies []Dependency         // Chart modifications
    JSONPatches []any                # JSON patches to apply
    StrategicMergePatches []any      # K8s strategic merge patches
    Transformers []any               # Kustomize transformers
    
    // Helm options
    Wait, WaitForJobs *bool          # Wait for deployment
    Timeout *int                     # Operation timeout
    Force, Atomic *bool              # Update strategy options
    ReuseValues *bool                # Preserve existing values
    
    // Validation options
    DisableValidation *bool          # Skip K8s validation
    DisableValidationOnInstall *bool # Skip validation on new install
    
    // Template-based values
    ValuesTemplate []any             # Template-rendered values
    SetValuesTemplate []SetValue     # Template-rendered set values
}
```

---

## Data Structures

### HelmState: `/home/user/helmfire/analysis/sources/helmfile/pkg/state/state.go` (Lines 116-133)

Central struct representing a helmfile state file:

```go
type HelmState struct {
    basePath string           // Helmfile directory
    FilePath string           // Absolute path to helmfile.yaml
    
    ReleaseSetSpec           // Embedded spec
    
    logger *zap.SugaredLogger
    fs *filesystem.FileSystem
    tempDir func(string, string) (string, error)
    
    valsRuntime vals.Evaluator  // For rendering values
    RenderedValues map[string]any // Helmfile-level values
}

type ReleaseSetSpec struct {
    DefaultHelmBinary, DefaultKustomizeBinary string
    DefaultValues []any                       // Helmfile-level values
    Environments map[string]EnvironmentSpec   // Environment definitions
    
    Bases []string                            // Base helmfiles to inherit
    HelmDefaults HelmSpec                     // Default helm options
    Helmfiles []SubHelmfileSpec               // Subhelmfile includes
    
    OverrideKubeContext, OverrideNamespace, OverrideChart string  // Overrides
    Repositories []RepositorySpec             // Helm repositories
    CommonLabels map[string]string            // Labels for all releases
    Releases []ReleaseSpec                    // Release definitions
    
    Hooks []event.Hook                        // Lifecycle hooks
    Templates map[string]TemplateSpec         // Reusable templates
}
```

### Template Data: `/home/user/helmfire/analysis/sources/helmfile/pkg/state/types.go` (Lines 1-72)

```go
type EnvironmentTemplateData struct {
    Environment environment.Environment
    Namespace string
    Values map[string]any
    StateValues *map[string]any
}

type releaseTemplateData struct {
    Environment environment.Environment
    Release releaseTemplateDataRelease  // Subset of ReleaseSpec
    Values map[string]any
    StateValues *map[string]any
    KubeContext, Namespace, Chart string
}
```

### Event/Hook System: `/home/user/helmfire/analysis/sources/helmfile/pkg/event/bus.go` (Lines 16-23)

```go
type Hook struct {
    Name string            // Hook name
    Events []string        // Events to trigger on (pre-sync, post-sync, etc.)
    Command string         // Shell command to execute
    Kubectl map[string]string  // kubectl apply options
    Args []string          // Command arguments
    ShowLogs bool          // Display output
}

type Bus struct {
    Runner helmexec.Runner
    Hooks []Hook
    BasePath, StateFilePath, Namespace, Chart string
    Env environment.Environment
    Fs *filesystem.FileSystem
    Logger *zap.SugaredLogger
}
```

### Configuration: `/home/user/helmfire/analysis/sources/helmfile/pkg/config/`

**GlobalOptions:** Stores CLI flags
- File, Environment, Namespace, Chart, Selector
- HelmBinary, KustomizeBinary, Kubeconfig, KubeContext
- StateValuesSet, StateValuesFile, StateValuesSetString
- Debug, Quiet, Color, NoColor, LogLevel

**Command-Specific Options:**
- SyncOptions, ApplyOptions, DiffOptions, DestroyOptions, etc.
- Each provides specialized flags for their command

---

## Dependencies

### Key External Libraries

**Helm Integration:**
```
helm.sh/helm/v3 v3.19.0
  - pkg/action     - Helm actions (install, upgrade, delete, list)
  - pkg/chart      - Chart handling
  - pkg/cli        - CLI utilities
  - pkg/plugin     - Plugin management
```

**Kubernetes:**
```
k8s.io/apimachinery v0.34.1
  - For K8s API validation and manifest handling
```

**YAML & Templating:**
```
go.yaml.in/yaml/v2 v2.4.3
go.yaml.in/yaml/v3 v3.0.4
  - YAML parsing and encoding
  
github.com/helmfile/vals v0.42.4
  - Render templated values (ref, vault, sops, etc.)
  
github.com/zclconf/go-cty v1.17.0
github.com/hashicorp/hcl/v2 v2.24.0
  - HCL language support for advanced configs
```

**Charting & Templating:**
```
github.com/helmfile/chartify v0.25.0
  - Chart processing and transformation
  
github.com/Masterminds/sprig/v3 v3.3.0
  - Go template function library
```

**Dependency Graph:**
```
github.com/variantdev/dag v1.1.0
  - DAG (Directed Acyclic Graph) for release ordering
```

**CLI Framework:**
```
github.com/spf13/cobra v1.10.1
github.com/spf13/pflag v1.0.10
  - Command-line interface building
```

**Cloud & Remote:**
```
github.com/hashicorp/go-getter v1.8.3
  - Fetch remote files (S3, HTTP, Git, etc.)
  
github.com/aws/aws-sdk-go-v2/service/s3 v1.90.0
  - AWS S3 support
  
cloud.google.com/go/storage v1.57.0
  - Google Cloud Storage support
```

**Logging:**
```
go.uber.org/zap v1.27.0
  - Structured logging with performance
```

**Utilities:**
```
dario.cat/mergo v1.0.2
  - Deep merging of maps and structures
  
github.com/Masterminds/semver/v3 v3.4.0
  - Semantic versioning
  
github.com/gosuri/uitable v0.0.4
  - ASCII table formatting
```

**Testing:**
```
github.com/stretchr/testify v1.11.1
github.com/golang/mock v1.6.0
  - Testing frameworks
```

---

## Reusable Components for Helmfire

### 1. Core State Management (`pkg/state/`)

**Import Path:** `github.com/helmfile/helmfile/pkg/state`

**Key Exportable Structs:**

- `HelmState` - Complete helmfile state representation
- `ReleaseSpec` - Individual release configuration
- `ReleaseSetSpec` - Helmfile-level configuration
- `HelmSpec` - Default helm options
- `RepositorySpec` - Helm repository configuration
- `Hook` - Event hooks configuration

**Key Exportable Functions:**

- `NewCreator()` - Create state from YAML
- `(s *HelmState) SyncRepos()` - Update helm repositories
- `(s *HelmState) SyncReleases()` - Deploy releases
- `(s *HelmState) DiffReleases()` - Show diffs
- `(s *HelmState) PrepareCharts()` - Download and prepare charts
- `(s *HelmState) PlanReleases()` - Generate deployment DAG

**Usage Example:**
```go
import "github.com/helmfile/helmfile/pkg/state"

creator := state.NewCreator(
    logger, fs, valsRuntime, getHelmFunc,
    helmBinary, kustomizeBinary, remote, 
    enableLiveOutput, lockFile)

helmState, err := creator.Parse(yamlContent, baseDir, filePath)
helmState.SyncReleases(affectedReleases, helm, additionalValues, concurrency)
```

### 2. Helm Execution (`pkg/helmexec/`)

**Import Path:** `github.com/helmfile/helmfile/pkg/helmexec`

**Key Exportable Interfaces:**

```go
type Interface interface {
    // Chart operations
    Chart(ctx HelmContext, chart, version string) (*chart.Chart, error)
    ChartMetadata(ctx HelmContext, chart string) (*Metadata, error)
    
    // Release operations
    Template(ctx HelmContext, ...) ([]byte, error)
    TemplateChart(ctx HelmContext, ...) ([]byte, error)
    Diff(ctx HelmContext, ...) (bool, error)
    List(ctx HelmContext, ...) (map[string][]string, error)
    Fetch(ctx HelmContext, ...) ([]byte, error)
    Status(ctx HelmContext, ...) ([]byte, error)
    Delete(ctx HelmContext, ...) ([]byte, error)
    
    // Repo operations
    AddRepo(ctx HelmContext, ...) error
    UpdateRepo(ctx HelmContext, ...) error
}

type Runner interface {
    Execute(cmd string, args []string, 
            env map[string]string, 
            showOutput bool) ([]byte, error)
}
```

**Key Exportable Functions:**

- `New()` - Initialize helm executor with version detection
- `GetHelmVersion()` - Detect installed helm version
- `NewLogger()` - Create structured logger

**Usage Example:**
```go
import "github.com/helmfile/helmfile/pkg/helmexec"

helm, err := helmexec.New(
    helmBinary, options, logger,
    kubeconfig, kubeContext, runner)

status, err := helm.Status(ctx, release.Name, release.Namespace)
```

### 3. App/Orchestration (`pkg/app/`)

**Import Path:** `github.com/helmfile/helmfile/pkg/app`

**Key Exportable Structs:**

```go
type App struct {
    OverrideKubeContext, OverrideHelmBinary string
    Logger *zap.SugaredLogger
    Kubeconfig, Env, Namespace, Chart, Args string
    Selectors []string
    FileOrDir string
    // ...
}

type ConfigProvider interface {
    KubeContext() string
    HelmBinary() string
    Kubeconfig() string
    Env() string
    FileOrDir() string
    // ... plus command-specific config methods
}
```

**Key Exportable Functions:**

- `New()` - Create app instance
- `(a *App) Sync()` - Execute sync operation
- `(a *App) Apply()` - Execute apply operation
- `(a *App) Diff()` - Show differences
- `(a *App) Destroy()` - Delete releases
- `(a *App) List()` - List releases
- `(a *App) Status()` - Show release status

**Usage Example:**
```go
import "github.com/helmfile/helmfile/pkg/app"

appInstance := app.New(configProvider)
err := appInstance.Sync(syncConfig)
```

### 4. YAML & Values Loading

**Import Path:** `github.com/helmfile/helmfile/pkg/yaml`

```go
func NewDecoder(data []byte, strict bool) func(any) error
func Unmarshal(data []byte, v any) error
func Marshal(v any) ([]byte, error)
func NewEncoder(w io.Writer) Encoder
```

**Import Path:** `github.com/helmfile/helmfile/pkg/app`

- `desiredStateLoader` - Internal loader for state files
- Can be adapted for custom loading logic

### 5. Event/Hook System (`pkg/event/`)

**Import Path:** `github.com/helmfile/helmfile/pkg/event`

```go
type Hook struct {
    Name string
    Events []string
    Command string
    Kubectl map[string]string
    Args []string
    ShowLogs bool
}

type Bus struct {
    Runner helmexec.Runner
    Hooks []Hook
    BasePath, StateFilePath string
    Namespace, Chart string
    Env environment.Environment
    Fs *filesystem.FileSystem
    Logger *zap.SugaredLogger
}

func (bus *Bus) Trigger(evt string, evtErr error, 
                        context map[string]any) (bool, error)
```

**Available Events:**
- `presync`, `postsync`
- `preapply`, `postapply`
- `prediff`, `postdiff`
- `predestroy`, `postdestroy`
- `presyncrelease`, `postsyncrelease`
- `predeleterelease`, `postdeleterelease`

### 6. Environment & Values Processing

**Import Path:** `github.com/helmfile/helmfile/pkg/state`

- `EnvironmentSpec` - Environment configuration
- `LoadYAMLForEmbedding()` - Load and process value files
- Template rendering with `.Environment`, `.Values`, `.Release` contexts

**Import Path:** `github.com/helmfile/helmfile/pkg/environment`

- `Environment` struct for managing env values
- Value lookup and merging

### 7. File System Abstraction (`pkg/filesystem/`)

**Import Path:** `github.com/helmfile/helmfile/pkg/filesystem`

- `FileSystem` interface for file operations
- Allows mocking and testing
- Used throughout for file I/O

### 8. Concurrency Patterns (`pkg/state/state_run.go`)

**Scatter-Gather Pattern:**

```go
func (st *HelmState) scatterGather(
    concurrency int, 
    items int,
    produceInputs func(),
    receiveInputsAndProduceIntermediates func(int),
    aggregateIntermediates func())

func (st *HelmState) iterateOnReleases(
    helm helmexec.Interface,
    concurrency int,
    inputs []ReleaseSpec,
    do func(ReleaseSpec, int) error) []error
```

**Usage:** Parallel release processing with configurable concurrency limits

### 9. DAG & Release Planning

**Import Path:** `github.com/helmfile/helmfile/pkg/state`

- `(st *HelmState) PlanReleases()` - Build dependency graph
- Uses `variantdev/dag` for topological sorting
- Returns batches of releases that can be executed in parallel

**Key Methods:**
```go
type PlanOptions struct {
    Purpose string
    Reverse bool
    IncludeNeeds bool
    IncludeTransitiveNeeds bool
    SkipNeeds bool
    SelectedReleases []ReleaseSpec
}

func (st *HelmState) PlanReleases(opts PlanOptions) ([][]Release, error)
```

### 10. Configuration Management

**Import Path:** `github.com/helmfile/helmfile/pkg/config`

All configuration structs are exportable:
- `GlobalOptions`, `GlobalImpl` - Global CLI flags
- `SyncOptions`, `ApplyOptions`, `DiffOptions`, etc.
- Configuration validation and merging

---

## Architecture Patterns for Helmfire Integration

### 1. State Loading Pipeline

```
helmfile.yaml (+ environment values)
    ↓
StateCreator.Parse()
    ↓
HelmState struct (with rendered templates)
    ↓
App.ForEachState()
    ↓
Run (executor with state + helm)
    ↓
Output/Operations
```

### 2. Release Execution Flow

```
HelmState.Releases[]
    ↓
Filter by selectors/conditions
    ↓
Resolve "needs" dependencies (DAG)
    ↓
PlanReleases() → Batches
    ↓
For each batch (parallel):
    PrepareCharts()
    Trigger pre-sync hooks
    SyncReleases() → helm upgrade --install
    Trigger post-sync hooks
    ↓
Collect results
```

### 3. Modularity Advantages

**For Helmfire Implementation:**

1. **Can import state management** without app/CLI layers
2. **Can import helmexec** for direct helm command building
3. **Can use HelmState** for configuration representation
4. **Can leverage event system** for lifecycle hooks
5. **Can reuse environment rendering** for template processing
6. **Can extend Release definitions** for additional features

### 4. Key Integration Points

**State Watching Opportunity:**
- Monitor helmfile.yaml and values files
- On change: reload state with `StateCreator.Parse()`
- Recalculate DAG with `PlanReleases()`
- Trigger selective sync based on changes

**Drift Detection Opportunity:**
- Use `HelmState.DiffReleases()` for change detection
- Compare rendered templates with cluster state
- Implement continuous drift monitoring

**Enhanced Release Planning Opportunity:**
- Build on `PlanReleases()` DAG
- Add cost analysis for release changes
- Implement smart scheduling

---

## Critical Methods for Helmfire Extension

### State Rendering & Processing

- **`StateCreator.Parse()`** - Parse helmfile.yaml (Line 100 in create.go)
- **`desiredStateLoader.Load()`** - Load with overrides (Line 48 in desired_state_file_loader.go)
- **`HelmState.ExecuteTemplateExpressions()`** - Render release templates

### Release Execution

- **`HelmState.SyncReleases()`** - Main deployment logic (Line 975 in state.go)
- **`HelmState.DiffReleases()`** - Show changes (Line 2217 in state.go)
- **`HelmState.PrepareCharts()`** - Download and process (Line 1517 in state.go)
- **`HelmState.PlanReleases()`** - Build execution plan (state_run.go)

### Helm Command Building

- **`HelmExec.reformat()`** - Format helm arguments (Line 505 in state.go)
- **`HelmExec.Diff()`** - Generate diffs (helmexec/exec.go)
- **`HelmExec.Template()`** - Render manifests (helmexec/exec.go)
- **`HelmExec.TemplateChart()`** - Template with chart (helmexec/exec.go)

### Context & Configuration

- **`App.getHelm()`** - Get helm instance with context
- **`Run.withPreparedCharts()`** - Wrap operations with chart prep (Line 60 in run.go)
- **`Context.SyncReposOnce()`** - Cache repo updates (context.go)

---

## Summary for Helmfire Development

### Recommended Approach:

1. **Import `pkg/state`** for state loading and DAG planning
2. **Import `pkg/helmexec`** for helm command execution
3. **Import `pkg/app`** for orchestration patterns
4. **Create wrapper layer** for helmfire-specific features:
   - File watching (fswatcher/fsnotify)
   - Continuous drift detection
   - State change tracking
   - Release impact analysis

### Key Packages to Reuse:

| Package | Purpose | Complexity |
|---------|---------|-----------|
| `pkg/state` | Configuration and release management | High |
| `pkg/helmexec` | Helm command execution | Medium |
| `pkg/app` | Orchestration and CLI | Medium |
| `pkg/event` | Hook/event system | Low |
| `pkg/yaml` | YAML processing | Low |
| `pkg/environment` | Environment variables | Low |

### Architecture Strength:

- **Decoupled packages** - Each can be used independently
- **Interface-based** - Easy to mock and extend
- **Concurrent execution** - Built-in parallelism
- **Error handling** - Comprehensive error types
- **Logging** - Structured logging throughout

This architecture makes helmfile an excellent foundation for helmfire to build upon.


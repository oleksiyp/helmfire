module github.com/oleksiyp/helmfire

go 1.24

require (
	github.com/fsnotify/fsnotify v1.8.0
	github.com/helmfile/helmfile v0.169.1
	github.com/spf13/cobra v1.10.1
	go.uber.org/zap v1.27.0
	gopkg.in/yaml.v3 v3.0.4
	helm.sh/helm/v4 v4.0.0-20250101000000-000000000000
)

// Temporary replace directive until helm v4 is released
// This will be updated to actual version when available
replace helm.sh/helm/v4 => github.com/helm/helm v3.17.0+incompatible

package helmstate

// HelmfileSpec represents a simplified helmfile.yaml structure
type HelmfileSpec struct {
	Repositories []Repository `yaml:"repositories,omitempty"`
	Releases     []Release    `yaml:"releases"`
	Environments map[string]Environment `yaml:"environments,omitempty"`
}

// Repository represents a helm repository
type Repository struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	OCI      bool   `yaml:"oci,omitempty"`
}

// Release represents a helm release
type Release struct {
	Name      string                 `yaml:"name"`
	Namespace string                 `yaml:"namespace,omitempty"`
	Chart     string                 `yaml:"chart"`
	Version   string                 `yaml:"version,omitempty"`
	Values    []interface{}          `yaml:"values,omitempty"`
	Set       []SetValue             `yaml:"set,omitempty"`
	Wait      bool                   `yaml:"wait,omitempty"`
	Installed *bool                  `yaml:"installed,omitempty"`
	Labels    map[string]string      `yaml:"labels,omitempty"`
}

// SetValue represents a --set style value
type SetValue struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// Environment represents an environment configuration
type Environment struct {
	Values []interface{} `yaml:"values,omitempty"`
}

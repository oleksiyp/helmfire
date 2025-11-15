package helmstate

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Manager manages helmfile state
type Manager struct {
	FilePath    string
	Environment string
	Spec        *HelmfileSpec
}

// NewManager creates a new helmstate manager
func NewManager(filePath, environment string) *Manager {
	return &Manager{
		FilePath:    filePath,
		Environment: environment,
	}
}

// Load loads and parses the helmfile
func (m *Manager) Load() error {
	absPath, err := filepath.Abs(m.FilePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read helmfile: %w", err)
	}

	spec := &HelmfileSpec{}
	if err := yaml.Unmarshal(data, spec); err != nil {
		return fmt.Errorf("failed to parse helmfile: %w", err)
	}

	m.Spec = spec
	m.FilePath = absPath
	return nil
}

// GetReleases returns all releases
func (m *Manager) GetReleases() []Release {
	if m.Spec == nil {
		return nil
	}
	return m.Spec.Releases
}

// GetRepositories returns all repositories
func (m *Manager) GetRepositories() []Repository {
	if m.Spec == nil {
		return nil
	}
	return m.Spec.Repositories
}

// FilterReleases filters releases by selector
func (m *Manager) FilterReleases(selector map[string]string) []Release {
	if m.Spec == nil || len(selector) == 0 {
		return m.GetReleases()
	}

	var filtered []Release
	for _, release := range m.Spec.Releases {
		matches := true
		for key, value := range selector {
			if release.Labels[key] != value {
				matches = false
				break
			}
		}
		if matches {
			filtered = append(filtered, release)
		}
	}
	return filtered
}

// IsReleaseInstalled checks if a release should be installed
func (m *Manager) IsReleaseInstalled(release Release) bool {
	if release.Installed == nil {
		return true // default is installed
	}
	return *release.Installed
}

// DiffRelease runs helm diff for a release to detect drift
func (m *Manager) DiffRelease(release Release) (string, error) {
	namespace := release.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// Build helm diff command
	args := []string{
		"diff",
		"upgrade",
		release.Name,
		release.Chart,
		"--namespace", namespace,
		"--allow-unreleased",
	}

	// Add values files
	for _, valuesFile := range release.Values {
		if strVal, ok := valuesFile.(string); ok {
			args = append(args, "--values", strVal)
		}
	}

	// Execute helm diff
	cmd := exec.Command("helm", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Exit code 2 means there are differences (which is what we want to detect)
		// Exit code 0 means no differences
		// Other exit codes are actual errors
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 2 {
				// Differences detected - return the diff output
				return stdout.String(), nil
			}
		}
		return "", fmt.Errorf("helm diff failed: %w (stderr: %s)", err, stderr.String())
	}

	// No differences
	return "", nil
}

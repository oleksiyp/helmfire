package helmstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	manager := NewManager("helmfile.yaml", "dev")
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}
	if manager.FilePath != "helmfile.yaml" {
		t.Errorf("expected FilePath helmfile.yaml, got %s", manager.FilePath)
	}
	if manager.Environment != "dev" {
		t.Errorf("expected Environment dev, got %s", manager.Environment)
	}
}

func TestLoad(t *testing.T) {
	// Create a temporary helmfile
	tmpDir := t.TempDir()
	helmfilePath := filepath.Join(tmpDir, "helmfile.yaml")

	helmfileContent := `
repositories:
  - name: bitnami
    url: https://charts.bitnami.com/bitnami

releases:
  - name: nginx
    namespace: default
    chart: bitnami/nginx
    version: 13.2.0
    labels:
      app: web
    values:
      - values.yaml
    set:
      - name: replicaCount
        value: "3"
`

	if err := os.WriteFile(helmfilePath, []byte(helmfileContent), 0644); err != nil {
		t.Fatalf("failed to write test helmfile: %v", err)
	}

	manager := NewManager(helmfilePath, "")
	if err := manager.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if manager.Spec == nil {
		t.Fatal("expected Spec to be loaded")
	}

	// Check repositories
	repos := manager.GetRepositories()
	if len(repos) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(repos))
	}
	if repos[0].Name != "bitnami" {
		t.Errorf("expected repository name bitnami, got %s", repos[0].Name)
	}

	// Check releases
	releases := manager.GetReleases()
	if len(releases) != 1 {
		t.Fatalf("expected 1 release, got %d", len(releases))
	}
	if releases[0].Name != "nginx" {
		t.Errorf("expected release name nginx, got %s", releases[0].Name)
	}
	if releases[0].Chart != "bitnami/nginx" {
		t.Errorf("expected chart bitnami/nginx, got %s", releases[0].Chart)
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	manager := NewManager("/nonexistent/helmfile.yaml", "")
	err := manager.Load()
	if err == nil {
		t.Fatal("expected error loading nonexistent file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	helmfilePath := filepath.Join(tmpDir, "helmfile.yaml")

	invalidYAML := `
releases:
  - name: test
    invalid yaml content [[[
`

	if err := os.WriteFile(helmfilePath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write test helmfile: %v", err)
	}

	manager := NewManager(helmfilePath, "")
	err := manager.Load()
	if err == nil {
		t.Fatal("expected error loading invalid YAML")
	}
}

func TestFilterReleases(t *testing.T) {
	tmpDir := t.TempDir()
	helmfilePath := filepath.Join(tmpDir, "helmfile.yaml")

	helmfileContent := `
releases:
  - name: nginx
    chart: bitnami/nginx
    labels:
      app: web
      tier: frontend
  - name: postgres
    chart: bitnami/postgresql
    labels:
      app: db
      tier: backend
  - name: redis
    chart: bitnami/redis
    labels:
      app: cache
      tier: backend
`

	if err := os.WriteFile(helmfilePath, []byte(helmfileContent), 0644); err != nil {
		t.Fatalf("failed to write test helmfile: %v", err)
	}

	manager := NewManager(helmfilePath, "")
	if err := manager.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	tests := []struct {
		name     string
		selector map[string]string
		expected int
	}{
		{
			name:     "no selector returns all",
			selector: map[string]string{},
			expected: 3,
		},
		{
			name:     "filter by tier=backend",
			selector: map[string]string{"tier": "backend"},
			expected: 2,
		},
		{
			name:     "filter by app=web",
			selector: map[string]string{"app": "web"},
			expected: 1,
		},
		{
			name:     "filter by multiple labels",
			selector: map[string]string{"app": "db", "tier": "backend"},
			expected: 1,
		},
		{
			name:     "no matches",
			selector: map[string]string{"app": "nonexistent"},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := manager.FilterReleases(tt.selector)
			if len(filtered) != tt.expected {
				t.Errorf("expected %d releases, got %d", tt.expected, len(filtered))
			}
		})
	}
}

func TestIsReleaseInstalled(t *testing.T) {
	manager := NewManager("", "")

	tests := []struct {
		name     string
		release  Release
		expected bool
	}{
		{
			name: "nil installed field defaults to true",
			release: Release{
				Name:      "test",
				Installed: nil,
			},
			expected: true,
		},
		{
			name: "installed true",
			release: Release{
				Name:      "test",
				Installed: boolPtr(true),
			},
			expected: true,
		},
		{
			name: "installed false",
			release: Release{
				Name:      "test",
				Installed: boolPtr(false),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.IsReleaseInstalled(tt.release)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetReleasesWithNilSpec(t *testing.T) {
	manager := NewManager("", "")
	releases := manager.GetReleases()
	if releases != nil {
		t.Errorf("expected nil releases, got %v", releases)
	}
}

func TestGetRepositoriesWithNilSpec(t *testing.T) {
	manager := NewManager("", "")
	repos := manager.GetRepositories()
	if repos != nil {
		t.Errorf("expected nil repositories, got %v", repos)
	}
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}

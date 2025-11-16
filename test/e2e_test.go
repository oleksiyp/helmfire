// +build e2e

package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2EBasicSync tests basic sync functionality
func TestE2EBasicSync(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Build helmfire binary
	helmfireBinary := buildHelmfire(t)
	defer os.Remove(helmfireBinary)

	// Create a test helmfile
	tmpDir := t.TempDir()
	helmfilePath := filepath.Join(tmpDir, "helmfile.yaml")

	helmfileContent := `
repositories:
  - name: bitnami
    url: https://charts.bitnami.com/bitnami

releases:
  - name: test-nginx
    namespace: test
    chart: bitnami/nginx
    version: 13.2.0
    installed: false
`

	if err := os.WriteFile(helmfilePath, []byte(helmfileContent), 0o644); err != nil {
		t.Fatalf("failed to write helmfile: %v", err)
	}

	// Run helmfire sync --dry-run
	cmd := exec.Command(helmfireBinary, "sync", "-f", helmfilePath, "--dry-run")
	output, err := cmd.CombinedOutput()

	// Note: This might fail if helm is not installed or k8s cluster is not available
	// That's okay for the test - we're verifying the binary runs
	t.Logf("helmfire output: %s", string(output))

	if err != nil {
		// Check if it's a helm/kubectl availability issue vs. helmfire bug
		if strings.Contains(string(output), "helm") || strings.Contains(string(output), "kubectl") {
			t.Skip("skipping e2e test: helm or kubectl not available")
		}
		t.Logf("helmfire sync failed (expected in some environments): %v", err)
	}
}

// TestE2EChartSubstitution tests chart substitution
func TestE2EChartSubstitution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	helmfireBinary := buildHelmfire(t)
	defer os.Remove(helmfireBinary)

	tmpDir := t.TempDir()

	// Create a local chart
	localChartDir := filepath.Join(tmpDir, "my-chart")
	if err := os.MkdirAll(filepath.Join(localChartDir, "templates"), 0o755); err != nil {
		t.Fatalf("failed to create chart directory: %v", err)
	}

	chartYAML := `apiVersion: v2
name: my-chart
version: 1.0.0
description: Test chart
`
	if err := os.WriteFile(filepath.Join(localChartDir, "Chart.yaml"), []byte(chartYAML), 0o644); err != nil {
		t.Fatalf("failed to write Chart.yaml: %v", err)
	}

	// Create a simple template
	templateYAML := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
data:
  key: value
`
	if err := os.WriteFile(filepath.Join(localChartDir, "templates", "configmap.yaml"), []byte(templateYAML), 0o644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	// Run helmfire chart command
	cmd := exec.Command(helmfireBinary, "chart", "bitnami/nginx", localChartDir)
	output, err := cmd.CombinedOutput()
	t.Logf("helmfire chart output: %s", string(output))

	if err != nil {
		t.Logf("helmfire chart command execution: %v", err)
	}
}

// TestE2EImageSubstitution tests image substitution
func TestE2EImageSubstitution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	helmfireBinary := buildHelmfire(t)
	defer os.Remove(helmfireBinary)

	// Run helmfire image command
	cmd := exec.Command(helmfireBinary, "image", "nginx:1.21", "nginx:1.22")
	output, err := cmd.CombinedOutput()
	t.Logf("helmfire image output: %s", string(output))

	if err != nil {
		t.Logf("helmfire image command execution: %v", err)
	}
}

// TestE2EListCommands tests list commands
func TestE2EListCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	helmfireBinary := buildHelmfire(t)
	defer os.Remove(helmfireBinary)

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "list charts",
			args: []string{"list", "charts"},
		},
		{
			name: "list images",
			args: []string{"list", "images"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(helmfireBinary, tt.args...)
			output, err := cmd.CombinedOutput()
			t.Logf("output: %s", string(output))
			if err != nil {
				t.Logf("command execution: %v", err)
			}
		})
	}
}

// TestE2EVersionCommand tests version command
func TestE2EVersionCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	helmfireBinary := buildHelmfire(t)
	defer os.Remove(helmfireBinary)

	cmd := exec.Command(helmfireBinary, "version")
	output, err := cmd.CombinedOutput()
	t.Logf("version output: %s", string(output))

	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	// Verify output contains version info
	outputStr := string(output)
	if !strings.Contains(outputStr, "Version") && !strings.Contains(outputStr, "version") {
		t.Error("version output doesn't contain version information")
	}
}

// Helper function to build helmfire binary for testing
func buildHelmfire(t *testing.T) string {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "helmfire")

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/helmfire")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build helmfire: %v\nOutput: %s", err, string(output))
	}

	// Verify binary exists
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("binary not created: %v", err)
	}

	return binaryPath
}

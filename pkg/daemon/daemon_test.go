package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsDaemonRunning(t *testing.T) {
	// Create temporary PID file
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// Test: No PID file
	running, err := IsDaemonRunning(pidFile)
	if err != nil {
		t.Errorf("Expected no error when PID file doesn't exist, got: %v", err)
	}
	if running {
		t.Error("Expected daemon not running when PID file doesn't exist")
	}

	// Test: PID file with current process
	pid := os.Getpid()
	if err := os.WriteFile(pidFile, []byte(string(rune(pid))), 0644); err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	// Clean up
	defer os.Remove(pidFile)
}

func TestAPIClient(t *testing.T) {
	client := NewAPIClient("127.0.0.1:8080")
	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.baseURL != "http://127.0.0.1:8080" {
		t.Errorf("Expected baseURL to be http://127.0.0.1:8080, got: %s", client.baseURL)
	}

	if client.client == nil {
		t.Error("Expected non-nil HTTP client")
	}

	if client.client.Timeout != 10*time.Second {
		t.Errorf("Expected timeout to be 10s, got: %v", client.client.Timeout)
	}
}

func TestDaemonConfig(t *testing.T) {
	config := DaemonConfig{
		PIDFile:       "/tmp/test.pid",
		LogFile:       "/tmp/test.log",
		APIAddr:       "127.0.0.1:9090",
		HelmfilePath:  "helmfile.yaml",
		Environment:   "test",
		DriftInterval: 30 * time.Second,
		DriftAutoHeal: true,
		DriftWebhook:  "http://example.com/webhook",
	}

	if config.PIDFile != "/tmp/test.pid" {
		t.Errorf("Expected PIDFile to be /tmp/test.pid, got: %s", config.PIDFile)
	}

	if config.DriftInterval != 30*time.Second {
		t.Errorf("Expected DriftInterval to be 30s, got: %v", config.DriftInterval)
	}

	if !config.DriftAutoHeal {
		t.Error("Expected DriftAutoHeal to be true")
	}
}

func TestStatus(t *testing.T) {
	status := Status{
		Running:   true,
		PID:       12345,
		Uptime:    "1h30m",
		StartTime: time.Now(),
	}

	if !status.Running {
		t.Error("Expected status.Running to be true")
	}

	if status.PID != 12345 {
		t.Errorf("Expected PID to be 12345, got: %d", status.PID)
	}
}

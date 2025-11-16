package substitute

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.charts == nil || m.images == nil {
		t.Fatal("Manager maps not initialized")
	}
}

func TestAddChartSubstitution(t *testing.T) {
	m := NewManager()

	// Create a temporary chart directory
	tmpDir := t.TempDir()
	chartDir := filepath.Join(tmpDir, "test-chart")
	if err := os.Mkdir(chartDir, 0o755); err != nil {
		t.Fatalf("failed to create chart dir: %v", err)
	}

	// Create Chart.yaml
	chartYAML := filepath.Join(chartDir, "Chart.yaml")
	if err := os.WriteFile(chartYAML, []byte("name: test\nversion: 1.0.0\n"), 0o644); err != nil {
		t.Fatalf("failed to create Chart.yaml: %v", err)
	}

	// Test adding valid chart substitution
	err := m.AddChartSubstitution("myrepo/mychart", chartDir)
	if err != nil {
		t.Errorf("AddChartSubstitution failed: %v", err)
	}

	// Verify it was added
	path, ok := m.GetChartPath("myrepo/mychart")
	if !ok {
		t.Error("Chart substitution not found")
	}
	if path == "" {
		t.Error("Chart path is empty")
	}

	// Test adding invalid path
	err = m.AddChartSubstitution("other/chart", "/nonexistent/path")
	if err == nil {
		t.Error("Expected error for nonexistent path, got nil")
	}
}

func TestAddImageSubstitution(t *testing.T) {
	m := NewManager()

	err := m.AddImageSubstitution("nginx:1.21", "myregistry.io/nginx:custom")
	if err != nil {
		t.Errorf("AddImageSubstitution failed: %v", err)
	}

	// Verify it was added
	img, ok := m.GetImageReplacement("nginx:1.21")
	if !ok {
		t.Error("Image substitution not found")
	}
	if img != "myregistry.io/nginx:custom" {
		t.Errorf("Expected myregistry.io/nginx:custom, got %s", img)
	}

	// Test empty values
	err = m.AddImageSubstitution("", "something")
	if err == nil {
		t.Error("Expected error for empty original image")
	}

	err = m.AddImageSubstitution("something", "")
	if err == nil {
		t.Error("Expected error for empty replacement image")
	}
}

func TestRemoveChartSubstitution(t *testing.T) {
	m := NewManager()

	// Create temp chart
	tmpDir := t.TempDir()
	chartDir := filepath.Join(tmpDir, "test-chart")
	os.Mkdir(chartDir, 0o755)
	os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte("name: test\n"), 0o644)

	m.AddChartSubstitution("myrepo/mychart", chartDir)

	// Remove it
	err := m.RemoveChartSubstitution("myrepo/mychart")
	if err != nil {
		t.Errorf("RemoveChartSubstitution failed: %v", err)
	}

	// Verify it's gone
	_, ok := m.GetChartPath("myrepo/mychart")
	if ok {
		t.Error("Chart substitution still exists after removal")
	}

	// Test removing non-existent
	err = m.RemoveChartSubstitution("nonexistent")
	if err == nil {
		t.Error("Expected error removing non-existent substitution")
	}
}

func TestRemoveImageSubstitution(t *testing.T) {
	m := NewManager()

	m.AddImageSubstitution("nginx:1.21", "custom:latest")

	// Remove it
	err := m.RemoveImageSubstitution("nginx:1.21")
	if err != nil {
		t.Errorf("RemoveImageSubstitution failed: %v", err)
	}

	// Verify it's gone
	_, ok := m.GetImageReplacement("nginx:1.21")
	if ok {
		t.Error("Image substitution still exists after removal")
	}

	// Test removing non-existent
	err = m.RemoveImageSubstitution("nonexistent")
	if err == nil {
		t.Error("Expected error removing non-existent substitution")
	}
}

func TestListSubstitutions(t *testing.T) {
	m := NewManager()

	// Create temp chart
	tmpDir := t.TempDir()
	chartDir := filepath.Join(tmpDir, "test-chart")
	os.Mkdir(chartDir, 0o755)
	os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte("name: test\n"), 0o644)

	// Add substitutions
	m.AddChartSubstitution("repo1/chart1", chartDir)
	m.AddImageSubstitution("image1:tag1", "replacement1:tag1")
	m.AddImageSubstitution("image2:tag2", "replacement2:tag2")

	// Test list charts
	charts := m.ListChartSubstitutions()
	if len(charts) != 1 {
		t.Errorf("Expected 1 chart substitution, got %d", len(charts))
	}

	// Test list images
	images := m.ListImageSubstitutions()
	if len(images) != 2 {
		t.Errorf("Expected 2 image substitutions, got %d", len(images))
	}
}

func TestApplySubstitutions(t *testing.T) {
	m := NewManager()

	// Create temp chart
	tmpDir := t.TempDir()
	chartDir := filepath.Join(tmpDir, "test-chart")
	os.Mkdir(chartDir, 0o755)
	os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte("name: test\n"), 0o644)

	m.AddChartSubstitution("myrepo/mychart", chartDir)
	m.AddImageSubstitution("nginx:1.21", "custom:latest")

	// Test chart substitution
	newChart, applied := m.ApplyChartSubstitutions("myrepo/mychart")
	if !applied {
		t.Error("Chart substitution not applied")
	}
	if newChart == "myrepo/mychart" {
		t.Error("Chart was not substituted")
	}

	// Test non-matching chart
	newChart, applied = m.ApplyChartSubstitutions("other/chart")
	if applied {
		t.Error("Unexpected chart substitution applied")
	}
	if newChart != "other/chart" {
		t.Error("Chart was modified when it shouldn't be")
	}

	// Test image substitution
	newImage, applied := m.ApplyImageSubstitutions("nginx:1.21")
	if !applied {
		t.Error("Image substitution not applied")
	}
	if newImage != "custom:latest" {
		t.Errorf("Expected custom:latest, got %s", newImage)
	}

	// Test non-matching image
	newImage, applied = m.ApplyImageSubstitutions("other:tag")
	if applied {
		t.Error("Unexpected image substitution applied")
	}
	if newImage != "other:tag" {
		t.Error("Image was modified when it shouldn't be")
	}
}

func TestConcurrency(t *testing.T) {
	m := NewManager()

	// Create temp chart
	tmpDir := t.TempDir()
	chartDir := filepath.Join(tmpDir, "test-chart")
	os.Mkdir(chartDir, 0o755)
	os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte("name: test\n"), 0o644)

	// Test concurrent access
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			m.AddImageSubstitution("image", "replacement")
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			m.GetImageReplacement("image")
		}
		done <- true
	}()

	// Wait for both
	<-done
	<-done

	// Should not have panicked
}

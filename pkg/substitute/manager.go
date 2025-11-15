package substitute

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Manager handles chart and image substitutions
type Manager struct {
	charts map[string]string // original chart -> local path
	images map[string]string // original image -> replacement
	mu     sync.RWMutex
}

// ChartSubstitution represents a chart override
type ChartSubstitution struct {
	Original  string
	LocalPath string
}

// ImageSubstitution represents an image override
type ImageSubstitution struct {
	Original    string
	Replacement string
}

// NewManager creates a new substitution manager
func NewManager() *Manager {
	return &Manager{
		charts: make(map[string]string),
		images: make(map[string]string),
	}
}

// AddChartSubstitution registers a chart substitution
func (m *Manager) AddChartSubstitution(original, localPath string) error {
	// Validate local path exists
	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("invalid local path: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("local path does not exist: %w", err)
	}

	// Check if it's a valid chart directory
	chartYAML := filepath.Join(absPath, "Chart.yaml")
	if _, err := os.Stat(chartYAML); err != nil {
		return fmt.Errorf("not a valid chart directory (missing Chart.yaml): %s", absPath)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.charts[original] = absPath
	return nil
}

// AddImageSubstitution registers an image substitution
func (m *Manager) AddImageSubstitution(original, replacement string) error {
	// TODO: Validate image references
	if original == "" || replacement == "" {
		return fmt.Errorf("image references cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.images[original] = replacement
	return nil
}

// RemoveChartSubstitution removes a chart substitution
func (m *Manager) RemoveChartSubstitution(original string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.charts[original]; !ok {
		return fmt.Errorf("chart substitution not found: %s", original)
	}

	delete(m.charts, original)
	return nil
}

// RemoveImageSubstitution removes an image substitution
func (m *Manager) RemoveImageSubstitution(original string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.images[original]; !ok {
		return fmt.Errorf("image substitution not found: %s", original)
	}

	delete(m.images, original)
	return nil
}

// GetChartPath returns the local path for a chart, if substituted
func (m *Manager) GetChartPath(original string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path, ok := m.charts[original]
	return path, ok
}

// GetImageReplacement returns the replacement image, if substituted
func (m *Manager) GetImageReplacement(original string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	replacement, ok := m.images[original]
	return replacement, ok
}

// ListChartSubstitutions returns all chart substitutions
func (m *Manager) ListChartSubstitutions() []ChartSubstitution {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]ChartSubstitution, 0, len(m.charts))
	for original, localPath := range m.charts {
		result = append(result, ChartSubstitution{
			Original:  original,
			LocalPath: localPath,
		})
	}
	return result
}

// ListImageSubstitutions returns all image substitutions
func (m *Manager) ListImageSubstitutions() []ImageSubstitution {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]ImageSubstitution, 0, len(m.images))
	for original, replacement := range m.images {
		result = append(result, ImageSubstitution{
			Original:    original,
			Replacement: replacement,
		})
	}
	return result
}

// ApplyChartSubstitutions applies chart substitutions to a chart reference
// Returns the substituted path and true if a substitution was applied
func (m *Manager) ApplyChartSubstitutions(chart string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if localPath, ok := m.charts[chart]; ok {
		return localPath, true
	}
	return chart, false
}

// ApplyImageSubstitutions applies image substitutions to an image reference
// Returns the substituted image and true if a substitution was applied
func (m *Manager) ApplyImageSubstitutions(image string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if replacement, ok := m.images[image]; ok {
		return replacement, true
	}
	return image, false
}

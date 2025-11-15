package drift

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

// MockNotifier is a test notifier that collects reports
type MockNotifier struct {
	reports []DriftReport
}

func (m *MockNotifier) Notify(report DriftReport) error {
	m.reports = append(m.reports, report)
	return nil
}

func TestNewDetector(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	detector := NewDetector(nil, 30*time.Second, logger)

	if detector == nil {
		t.Fatal("expected non-nil detector")
	}

	if detector.interval != 30*time.Second {
		t.Errorf("expected interval 30s, got %v", detector.interval)
	}

	if detector.autoHeal {
		t.Error("expected autoHeal to be false by default")
	}
}

func TestAddNotifier(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	detector := NewDetector(nil, 30*time.Second, logger)

	notifier := &MockNotifier{}
	detector.AddNotifier(notifier)

	if len(detector.notifiers) != 1 {
		t.Errorf("expected 1 notifier, got %d", len(detector.notifiers))
	}
}

func TestEnableAutoHeal(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	detector := NewDetector(nil, 30*time.Second, logger)

	healFunc := func(releaseName string) error {
		return nil
	}

	detector.EnableAutoHeal(true, healFunc)

	if !detector.autoHeal {
		t.Error("expected autoHeal to be true")
	}

	if detector.healFunc == nil {
		t.Error("expected healFunc to be set")
	}
}

func TestDetectorStartStop(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	detector := NewDetector(nil, 1*time.Hour, logger) // Long interval to prevent actual checks

	ctx := context.Background()

	// Start detector
	if err := detector.Start(ctx); err != nil {
		t.Fatalf("failed to start detector: %v", err)
	}

	if !detector.running {
		t.Error("expected detector to be running")
	}

	// Try to start again (should fail)
	if err := detector.Start(ctx); err == nil {
		t.Error("expected error when starting already running detector")
	}

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Stop detector
	if err := detector.Stop(); err != nil {
		t.Fatalf("failed to stop detector: %v", err)
	}

	if detector.running {
		t.Error("expected detector to be stopped")
	}

	// Try to stop again (should fail)
	if err := detector.Stop(); err == nil {
		t.Error("expected error when stopping already stopped detector")
	}
}

func TestClassifyDrift(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	detector := NewDetector(nil, 30*time.Second, logger)

	driftType := detector.classifyDrift("some diff content")
	if driftType != DriftTypeConfiguration {
		t.Errorf("expected DriftTypeConfiguration, got %s", driftType)
	}
}

func TestCalculateSeverity(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	detector := NewDetector(nil, 30*time.Second, logger)

	tests := []struct {
		name     string
		diff     string
		expected Severity
	}{
		{"small diff", "small change", SeverityLow},
		{"medium diff", string(make([]byte, 500)), SeverityMedium},
		{"large diff", string(make([]byte, 2000)), SeverityHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity := detector.calculateSeverity(tt.diff)
			if severity != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, severity)
			}
		})
	}
}

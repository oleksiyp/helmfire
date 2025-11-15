package drift

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestStdoutNotifier(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	notifier := NewStdoutNotifier(logger)

	report := DriftReport{
		Timestamp:   time.Now(),
		ReleaseName: "test-release",
		Namespace:   "default",
		DriftType:   DriftTypeConfiguration,
		Severity:    SeverityMedium,
		Details:     "Test drift",
		Diff:        "some diff output",
		Healed:      false,
	}

	if err := notifier.Notify(report); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWebhookNotifier(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Decode body
		var report DriftReport
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}

		// Verify report
		if report.ReleaseName != "test-release" {
			t.Errorf("expected test-release, got %s", report.ReleaseName)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	notifier := NewWebhookNotifier(server.URL, logger)

	report := DriftReport{
		Timestamp:   time.Now(),
		ReleaseName: "test-release",
		Namespace:   "default",
		DriftType:   DriftTypeConfiguration,
		Severity:    SeverityMedium,
		Details:     "Test drift",
		Diff:        "some diff output",
		Healed:      false,
	}

	if err := notifier.Notify(report); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWebhookNotifier_Error(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	notifier := NewWebhookNotifier(server.URL, logger)

	report := DriftReport{
		Timestamp:   time.Now(),
		ReleaseName: "test-release",
		Namespace:   "default",
		DriftType:   DriftTypeConfiguration,
		Severity:    SeverityMedium,
		Details:     "Test drift",
		Diff:        "some diff output",
		Healed:      false,
	}

	if err := notifier.Notify(report); err == nil {
		t.Error("expected error for non-2xx status code")
	}
}

func TestFileNotifier(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	notifier := NewFileNotifier("/tmp/drift.log", logger)

	report := DriftReport{
		Timestamp:   time.Now(),
		ReleaseName: "test-release",
		Namespace:   "default",
		DriftType:   DriftTypeConfiguration,
		Severity:    SeverityMedium,
		Details:     "Test drift",
		Diff:        "some diff output",
		Healed:      false,
	}

	// This is a placeholder implementation, so it should not error
	if err := notifier.Notify(report); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

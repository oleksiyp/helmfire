package drift

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// StdoutNotifier outputs drift reports to standard output
type StdoutNotifier struct {
	logger *zap.Logger
}

// NewStdoutNotifier creates a new stdout notifier
func NewStdoutNotifier(logger *zap.Logger) *StdoutNotifier {
	return &StdoutNotifier{
		logger: logger,
	}
}

// Notify outputs the drift report to stdout
func (n *StdoutNotifier) Notify(report DriftReport) error {
	icon := "⚠️"
	if report.Healed {
		icon = "✅"
	}

	fmt.Printf("\n%s DRIFT DETECTED %s\n", icon, icon)
	fmt.Printf("Timestamp:    %s\n", report.Timestamp.Format(time.RFC3339))
	fmt.Printf("Release:      %s\n", report.ReleaseName)
	fmt.Printf("Namespace:    %s\n", report.Namespace)
	fmt.Printf("Type:         %s\n", report.DriftType)
	fmt.Printf("Severity:     %s\n", report.Severity)
	fmt.Printf("Details:      %s\n", report.Details)
	if report.Healed {
		fmt.Printf("Status:       Auto-healed\n")
	}
	fmt.Printf("\nDiff:\n%s\n", report.Diff)
	fmt.Printf("═══════════════════════════════════════════════════\n\n")

	n.logger.Warn("drift detected",
		zap.String("release", report.ReleaseName),
		zap.String("namespace", report.Namespace),
		zap.String("type", string(report.DriftType)),
		zap.String("severity", string(report.Severity)),
		zap.Bool("healed", report.Healed))

	return nil
}

// WebhookNotifier sends drift reports to a webhook URL
type WebhookNotifier struct {
	webhookURL string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewWebhookNotifier creates a new webhook notifier
func NewWebhookNotifier(webhookURL string, logger *zap.Logger) *WebhookNotifier {
	return &WebhookNotifier{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// Notify sends the drift report to the configured webhook
func (n *WebhookNotifier) Notify(report DriftReport) error {
	payload, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal drift report: %w", err)
	}

	req, err := http.NewRequest("POST", n.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-2xx status: %d", resp.StatusCode)
	}

	n.logger.Debug("webhook notification sent",
		zap.String("url", n.webhookURL),
		zap.String("release", report.ReleaseName),
		zap.Int("statusCode", resp.StatusCode))

	return nil
}

// FileNotifier writes drift reports to a file
type FileNotifier struct {
	filePath string
	logger   *zap.Logger
}

// NewFileNotifier creates a new file notifier
func NewFileNotifier(filePath string, logger *zap.Logger) *FileNotifier {
	return &FileNotifier{
		filePath: filePath,
		logger:   logger,
	}
}

// Notify appends the drift report to the configured file
func (n *FileNotifier) Notify(report DriftReport) error {
	// Implementation for file-based notification
	// For now, this is a placeholder - could be enhanced to write JSON lines to a file
	n.logger.Info("file notification",
		zap.String("file", n.filePath),
		zap.String("release", report.ReleaseName))
	return nil
}

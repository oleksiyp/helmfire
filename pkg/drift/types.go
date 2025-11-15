package drift

import (
	"time"
)

// DriftType represents the category of drift detected
type DriftType string

const (
	DriftTypeConfiguration DriftType = "configuration"
	DriftTypeResource      DriftType = "resource"
	DriftTypeImage         DriftType = "image"
	DriftTypeDeletion      DriftType = "deletion"
)

// Severity indicates the importance of the drift
type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

// DriftReport describes detected drift in a release
type DriftReport struct {
	Timestamp   time.Time `json:"timestamp"`
	ReleaseName string    `json:"releaseName"`
	Namespace   string    `json:"namespace"`
	DriftType   DriftType `json:"driftType"`
	Severity    Severity  `json:"severity"`
	Details     string    `json:"details"`
	Diff        string    `json:"diff"`
	Healed      bool      `json:"healed"`
}

// Notifier defines the interface for drift notification mechanisms
type Notifier interface {
	Notify(report DriftReport) error
}

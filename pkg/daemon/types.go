package daemon

import (
	"context"
	"os"
	"time"

	"github.com/oleksiyp/helmfire/pkg/drift"
	"github.com/oleksiyp/helmfire/pkg/helmstate"
	"github.com/oleksiyp/helmfire/pkg/substitute"
	"go.uber.org/zap"
)

// Daemon manages background helmfire process
type Daemon struct {
	pidFile      string
	logFile      string
	apiAddr      string
	apiServer    *APIServer
	substitutor  *substitute.Manager
	manager      *helmstate.Manager
	detector     *drift.Detector
	logger       *zap.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	shutdownCh   chan os.Signal
	startTime    time.Time
}

// DaemonConfig configures the daemon
type DaemonConfig struct {
	PIDFile         string
	LogFile         string
	APIAddr         string
	HelmfilePath    string
	Environment     string
	DriftInterval   time.Duration
	DriftAutoHeal   bool
	DriftWebhook    string
}

// Status represents daemon status
type Status struct {
	Running             bool      `json:"running"`
	PID                 int       `json:"pid,omitempty"`
	Uptime              string    `json:"uptime,omitempty"`
	StartTime           time.Time `json:"startTime,omitempty"`
	LastSync            time.Time `json:"lastSync,omitempty"`
	ActiveSubstitutions struct {
		Charts int `json:"charts"`
		Images int `json:"images"`
	} `json:"activeSubstitutions"`
}

// SubstitutionsResponse represents API response for substitutions
type SubstitutionsResponse struct {
	Charts []ChartSubstitution `json:"charts"`
	Images []ImageSubstitution `json:"images"`
}

// ChartSubstitution represents a chart override
type ChartSubstitution struct {
	Original  string `json:"original"`
	LocalPath string `json:"localPath"`
}

// ImageSubstitution represents an image override
type ImageSubstitution struct {
	Original    string `json:"original"`
	Replacement string `json:"replacement"`
}

// AddChartRequest represents request to add chart substitution
type AddChartRequest struct {
	Original  string `json:"original"`
	LocalPath string `json:"localPath"`
}

// AddImageRequest represents request to add image substitution
type AddImageRequest struct {
	Original    string `json:"original"`
	Replacement string `json:"replacement"`
}

// RemoveChartRequest represents request to remove chart substitution
type RemoveChartRequest struct {
	Original string `json:"original"`
}

// RemoveImageRequest represents request to remove image substitution
type RemoveImageRequest struct {
	Original string `json:"original"`
}

// SyncRequest represents request to trigger sync
type SyncRequest struct {
	Releases []string `json:"releases,omitempty"`
	DryRun   bool     `json:"dryRun"`
}

// ErrorResponse represents API error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse represents API success response
type SuccessResponse struct {
	Message string `json:"message"`
}

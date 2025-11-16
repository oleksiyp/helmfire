// Package drift provides drift detection and auto-healing capabilities.
package drift

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/oleksiyp/helmfire/pkg/helmstate"
	"go.uber.org/zap"
)

// Detector monitors for configuration drift between desired and actual state
type Detector struct {
	manager    *helmstate.Manager
	interval   time.Duration
	autoHeal   bool
	notifiers  []Notifier
	logger     *zap.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	running    bool
	healFunc   func(releaseName string) error
}

// NewDetector creates a new drift detector
func NewDetector(manager *helmstate.Manager, interval time.Duration, logger *zap.Logger) *Detector {
	return &Detector{
		manager:   manager,
		interval:  interval,
		autoHeal:  false,
		notifiers: make([]Notifier, 0),
		logger:    logger,
		running:   false,
	}
}

// AddNotifier adds a notification handler for drift reports
func (d *Detector) AddNotifier(n Notifier) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.notifiers = append(d.notifiers, n)
}

// EnableAutoHeal enables or disables automatic healing of drift
func (d *Detector) EnableAutoHeal(enable bool, healFunc func(string) error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.autoHeal = enable
	d.healFunc = healFunc
}

// Start begins the drift detection monitoring loop
func (d *Detector) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return fmt.Errorf("detector already running")
	}

	d.ctx, d.cancel = context.WithCancel(ctx)
	d.running = true
	d.mu.Unlock()

	d.logger.Info("starting drift detector",
		zap.Duration("interval", d.interval),
		zap.Bool("autoHeal", d.autoHeal))

	d.wg.Add(1)
	go d.run()

	return nil
}

// Stop halts the drift detection monitoring
func (d *Detector) Stop() error {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return fmt.Errorf("detector not running")
	}
	d.mu.Unlock()

	d.logger.Info("stopping drift detector")
	d.cancel()
	d.wg.Wait()

	d.mu.Lock()
	d.running = false
	d.mu.Unlock()

	return nil
}

// run is the main monitoring loop
func (d *Detector) run() {
	defer d.wg.Done()

	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	// Run initial check
	d.checkDrift()

	for {
		select {
		case <-d.ctx.Done():
			d.logger.Info("drift detector context cancelled")
			return
		case <-ticker.C:
			d.checkDrift()
		}
	}
}

// checkDrift performs a single drift detection check across all releases
func (d *Detector) checkDrift() {
	d.logger.Debug("checking for drift")

	if d.manager == nil {
		d.logger.Debug("no manager configured")
		return
	}

	releases := d.manager.GetReleases()
	if len(releases) == 0 {
		d.logger.Debug("no releases to check for drift")
		return
	}

	for _, release := range releases {
		// Skip releases that are not installed
		if !d.manager.IsReleaseInstalled(release) {
			continue
		}

		report := d.checkReleaseDrift(release)
		if report != nil {
			d.handleDriftReport(*report)
		}
	}
}

// checkReleaseDrift checks a single release for drift
func (d *Detector) checkReleaseDrift(release helmstate.Release) *DriftReport {
	d.logger.Debug("checking release for drift",
		zap.String("release", release.Name),
		zap.String("namespace", release.Namespace))

	// Get the diff output
	diff, err := d.manager.DiffRelease(release)
	if err != nil {
		d.logger.Error("failed to diff release",
			zap.String("release", release.Name),
			zap.Error(err))
		return nil
	}

	// If diff is empty, no drift detected
	if diff == "" {
		d.logger.Debug("no drift detected",
			zap.String("release", release.Name))
		return nil
	}

	// Drift detected - create report
	d.logger.Info("drift detected",
		zap.String("release", release.Name),
		zap.String("namespace", release.Namespace))

	return &DriftReport{
		Timestamp:   time.Now(),
		ReleaseName: release.Name,
		Namespace:   release.Namespace,
		DriftType:   d.classifyDrift(diff),
		Severity:    d.calculateSeverity(diff),
		Details:     "Configuration drift detected",
		Diff:        diff,
		Healed:      false,
	}
}

// classifyDrift determines the type of drift from the diff output
func (d *Detector) classifyDrift(diff string) DriftType {
	// Simple classification based on diff content
	// This could be enhanced with more sophisticated analysis
	return DriftTypeConfiguration
}

// calculateSeverity determines the severity of the drift
func (d *Detector) calculateSeverity(diff string) Severity {
	// Simple severity calculation
	// Could be enhanced to analyze the actual changes
	diffLen := len(diff)
	if diffLen > 1000 {
		return SeverityHigh
	} else if diffLen > 100 {
		return SeverityMedium
	}
	return SeverityLow
}

// handleDriftReport processes a drift report
func (d *Detector) handleDriftReport(report DriftReport) {
	// Notify all registered notifiers
	d.mu.RLock()
	notifiers := make([]Notifier, len(d.notifiers))
	copy(notifiers, d.notifiers)
	autoHeal := d.autoHeal
	healFunc := d.healFunc
	d.mu.RUnlock()

	for _, notifier := range notifiers {
		if err := notifier.Notify(report); err != nil {
			d.logger.Error("failed to notify",
				zap.String("release", report.ReleaseName),
				zap.Error(err))
		}
	}

	// Auto-heal if enabled
	if autoHeal && healFunc != nil {
		d.logger.Info("attempting auto-heal",
			zap.String("release", report.ReleaseName))

		if err := healFunc(report.ReleaseName); err != nil {
			d.logger.Error("auto-heal failed",
				zap.String("release", report.ReleaseName),
				zap.Error(err))
		} else {
			d.logger.Info("auto-heal successful",
				zap.String("release", report.ReleaseName))

			// Update report and re-notify
			report.Healed = true
			report.Details = "Configuration drift detected and auto-healed"
			for _, notifier := range notifiers {
				if err := notifier.Notify(report); err != nil {
					d.logger.Error("failed to notify heal success",
						zap.String("release", report.ReleaseName),
						zap.Error(err))
				}
			}
		}
	}
}

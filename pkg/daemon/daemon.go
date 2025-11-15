package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/oleksiyp/helmfire/pkg/drift"
	"github.com/oleksiyp/helmfire/pkg/helmstate"
	"github.com/oleksiyp/helmfire/pkg/substitute"
	"go.uber.org/zap"
)

const (
	DefaultPIDFile = "/tmp/helmfire.pid"
	DefaultLogFile = "/tmp/helmfire.log"
	DefaultAPIAddr = "127.0.0.1:8080"
)

// NewDaemon creates a new daemon instance
func NewDaemon(config DaemonConfig, logger *zap.Logger) (*Daemon, error) {
	// Set defaults
	if config.PIDFile == "" {
		config.PIDFile = DefaultPIDFile
	}
	if config.LogFile == "" {
		config.LogFile = DefaultLogFile
	}
	if config.APIAddr == "" {
		config.APIAddr = DefaultAPIAddr
	}

	ctx, cancel := context.WithCancel(context.Background())

	d := &Daemon{
		pidFile:    config.PIDFile,
		logFile:    config.LogFile,
		apiAddr:    config.APIAddr,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
		shutdownCh: make(chan os.Signal, 1),
		startTime:  time.Now(),
	}

	// Initialize substitutor
	d.substitutor = substitute.NewManager()

	// Initialize helmfile manager
	d.manager = helmstate.NewManager(config.HelmfilePath, config.Environment)
	if err := d.manager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load helmfile: %w", err)
	}

	// Initialize drift detector if configured
	if config.DriftInterval > 0 {
		d.detector = drift.NewDetector(d.manager, config.DriftInterval, logger)
		d.detector.AddNotifier(drift.NewStdoutNotifier(logger))

		if config.DriftWebhook != "" {
			d.detector.AddNotifier(drift.NewWebhookNotifier(config.DriftWebhook, logger))
		}

		if config.DriftAutoHeal {
			// Auto-heal function will be set when we have access to executor
			d.detector.EnableAutoHeal(true, nil)
		}
	}

	// Initialize API server
	d.apiServer = NewAPIServer(d.apiAddr, d, logger)

	return d, nil
}

// Start starts the daemon
func (d *Daemon) Start() error {
	// Check if already running
	if running, err := d.IsRunning(); err == nil && running {
		return fmt.Errorf("daemon already running (PID file: %s)", d.pidFile)
	}

	// Write PID file
	if err := d.writePIDFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	d.logger.Info("daemon starting",
		zap.String("pidFile", d.pidFile),
		zap.String("logFile", d.logFile),
		zap.String("apiAddr", d.apiAddr))

	// Start API server
	if err := d.apiServer.Start(); err != nil {
		d.removePIDFile()
		return fmt.Errorf("failed to start API server: %w", err)
	}

	// Start drift detector if configured
	if d.detector != nil {
		if err := d.detector.Start(d.ctx); err != nil {
			d.apiServer.Stop()
			d.removePIDFile()
			return fmt.Errorf("failed to start drift detector: %w", err)
		}
		d.logger.Info("drift detector started")
	}

	// Setup signal handling
	signal.Notify(d.shutdownCh, os.Interrupt, syscall.SIGTERM)

	d.logger.Info("daemon started successfully")
	return nil
}

// Wait waits for the daemon to be stopped
func (d *Daemon) Wait() error {
	// Wait for shutdown signal
	sig := <-d.shutdownCh
	d.logger.Info("received shutdown signal", zap.String("signal", sig.String()))

	return d.Stop()
}

// Stop stops the daemon
func (d *Daemon) Stop() error {
	d.logger.Info("daemon stopping")

	// Cancel context
	d.cancel()

	// Stop drift detector
	if d.detector != nil {
		if err := d.detector.Stop(); err != nil {
			d.logger.Error("failed to stop drift detector", zap.Error(err))
		}
	}

	// Stop API server
	if err := d.apiServer.Stop(); err != nil {
		d.logger.Error("failed to stop API server", zap.Error(err))
	}

	// Remove PID file
	if err := d.removePIDFile(); err != nil {
		d.logger.Error("failed to remove PID file", zap.Error(err))
	}

	d.logger.Info("daemon stopped")
	return nil
}

// IsRunning checks if the daemon is running
func (d *Daemon) IsRunning() (bool, error) {
	return IsDaemonRunning(d.pidFile)
}

// GetPID returns the daemon PID
func (d *Daemon) GetPID() (int, error) {
	data, err := os.ReadFile(d.pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("daemon not running (PID file not found)")
		}
		return 0, err
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %s", pidStr)
	}

	return pid, nil
}

// GetStatus returns the daemon status
func (d *Daemon) GetStatus() Status {
	status := Status{
		Running:   true,
		PID:       os.Getpid(),
		StartTime: d.startTime,
		Uptime:    time.Since(d.startTime).Round(time.Second).String(),
	}

	// Get substitution counts
	charts := d.substitutor.ListChartSubstitutions()
	images := d.substitutor.ListImageSubstitutions()
	status.ActiveSubstitutions.Charts = len(charts)
	status.ActiveSubstitutions.Images = len(images)

	return status
}

// GetSubstitutor returns the substitution manager
func (d *Daemon) GetSubstitutor() *substitute.Manager {
	return d.substitutor
}

// GetManager returns the helmfile manager
func (d *Daemon) GetManager() *helmstate.Manager {
	return d.manager
}

// GetDetector returns the drift detector
func (d *Daemon) GetDetector() *drift.Detector {
	return d.detector
}

// writePIDFile writes the current PID to the PID file
func (d *Daemon) writePIDFile() error {
	pid := os.Getpid()
	return os.WriteFile(d.pidFile, []byte(fmt.Sprintf("%d\n", pid)), 0644)
}

// removePIDFile removes the PID file
func (d *Daemon) removePIDFile() error {
	return os.Remove(d.pidFile)
}

// IsDaemonRunning checks if a daemon is running based on PID file
func IsDaemonRunning(pidFile string) (bool, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return false, fmt.Errorf("invalid PID in file: %s", pidStr)
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, nil
	}

	// Send signal 0 to check if process is running
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return false, nil
	}

	return true, nil
}

// StopDaemon stops a running daemon
func StopDaemon(pidFile string) error {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("daemon not running (PID file not found)")
		}
		return err
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("invalid PID in file: %s", pidStr)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}

	// Send SIGTERM
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait for process to exit (with timeout)
	for i := 0; i < 30; i++ {
		err := process.Signal(syscall.Signal(0))
		if err != nil {
			// Process no longer exists
			os.Remove(pidFile)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	// If still running, send SIGKILL
	if err := process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}

	os.Remove(pidFile)
	return nil
}

// GetDaemonStatus returns the status of a daemon
func GetDaemonStatus(pidFile, apiAddr string) (*Status, error) {
	running, err := IsDaemonRunning(pidFile)
	if err != nil {
		return nil, err
	}

	if !running {
		return &Status{Running: false}, nil
	}

	// Get status from API
	client := NewAPIClient(apiAddr)
	return client.GetStatus()
}

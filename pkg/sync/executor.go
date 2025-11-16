package sync

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/oleksiyp/helmfire/pkg/helmstate"
	"github.com/oleksiyp/helmfire/pkg/substitute"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Executor handles release synchronization
type Executor struct {
	helmBinary  string
	namespace   string
	kubeContext string
	logger      *zap.Logger
	substitutor *substitute.Manager
	dryRun      bool
}

// NewExecutor creates a new sync executor
func NewExecutor(logger *zap.Logger, substitutor *substitute.Manager) *Executor {
	return &Executor{
		helmBinary:  "helm",
		logger:      logger,
		substitutor: substitutor,
	}
}

// SetDryRun enables or disables dry-run mode
func (e *Executor) SetDryRun(dryRun bool) {
	e.dryRun = dryRun
}

// SetNamespace sets the default namespace
func (e *Executor) SetNamespace(namespace string) {
	e.namespace = namespace
}

// SetKubeContext sets the kubectl context
func (e *Executor) SetKubeContext(context string) {
	e.kubeContext = context
}

// SyncRepositories adds/updates helm repositories
func (e *Executor) SyncRepositories(repos []helmstate.Repository) error {
	for _, repo := range repos {
		e.logger.Info("syncing repository", zap.String("name", repo.Name), zap.String("url", repo.URL))

		args := []string{"repo", "add", repo.Name, repo.URL}
		if repo.Username != "" {
			args = append(args, "--username", repo.Username)
		}
		if repo.Password != "" {
			args = append(args, "--password", repo.Password)
		}

		if err := e.runHelm(args...); err != nil {
			return fmt.Errorf("failed to add repository %s: %w", repo.Name, err)
		}
	}

	// Update all repositories
	if len(repos) > 0 {
		e.logger.Info("updating repositories")
		if err := e.runHelm("repo", "update"); err != nil {
			return fmt.Errorf("failed to update repositories: %w", err)
		}
	}

	return nil
}

// SyncRelease synchronizes a single release
func (e *Executor) SyncRelease(release helmstate.Release) error {
	// Apply chart substitution
	chart := release.Chart
	if localPath, ok := e.substitutor.GetChartPath(chart); ok {
		e.logger.Info("using local chart",
			zap.String("original", chart),
			zap.String("local", localPath))
		chart = localPath
	}

	// Determine namespace
	namespace := release.Namespace
	if namespace == "" {
		namespace = e.namespace
	}
	if namespace == "" {
		namespace = "default"
	}

	e.logger.Info("syncing release",
		zap.String("name", release.Name),
		zap.String("namespace", namespace),
		zap.String("chart", chart))

	// Build helm upgrade --install command
	args := []string{"upgrade", "--install", release.Name, chart}

	if namespace != "" {
		args = append(args, "--namespace", namespace)
		args = append(args, "--create-namespace")
	}

	if e.kubeContext != "" {
		args = append(args, "--kube-context", e.kubeContext)
	}

	if release.Version != "" {
		args = append(args, "--version", release.Version)
	}

	if release.Wait {
		args = append(args, "--wait")
	}

	// Add values files
	for _, val := range release.Values {
		if valStr, ok := val.(string); ok {
			args = append(args, "-f", valStr)
		}
	}

	// Add --set values
	for _, set := range release.Set {
		args = append(args, "--set", fmt.Sprintf("%s=%s", set.Name, set.Value))
	}

	if e.dryRun {
		args = append(args, "--dry-run")
	}

	// Check if we have image substitutions - if so, use post-renderer
	if len(e.substitutor.ListImageSubstitutions()) > 0 {
		// Create temporary post-renderer script
		postRenderer, err := e.createImagePostRenderer()
		if err != nil {
			return fmt.Errorf("failed to create post-renderer: %w", err)
		}
		defer os.Remove(postRenderer)

		args = append(args, "--post-renderer", postRenderer)
	}

	return e.runHelm(args...)
}

// createImagePostRenderer creates a temporary script for image substitution
func (e *Executor) createImagePostRenderer() (string, error) {
	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, "helmfire-post-renderer.sh")

	// Build substitution map
	substitutions := e.substitutor.ListImageSubstitutions()
	sedCommands := make([]string, 0, len(substitutions))

	for _, sub := range substitutions {
		// Escape special characters for sed
		original := strings.ReplaceAll(sub.Original, "/", "\\/")
		replacement := strings.ReplaceAll(sub.Replacement, "/", "\\/")
		sedCommands = append(sedCommands, fmt.Sprintf("s/image: %s/image: %s/g", original, replacement))
	}

	script := fmt.Sprintf(`#!/bin/bash
cat <&0 | sed '%s'
`, strings.Join(sedCommands, ";"))

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return "", err
	}

	return scriptPath, nil
}

// CreateImagePostRendererForBenchmark is a public wrapper for benchmarking
func (e *Executor) CreateImagePostRendererForBenchmark() (string, error) {
	return e.createImagePostRenderer()
}

// runHelm executes a helm command
func (e *Executor) runHelm(args ...string) error {
	cmd := exec.Command(e.helmBinary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	e.logger.Debug("executing helm command", zap.Strings("args", args))

	if err := cmd.Run(); err != nil {
		e.logger.Error("helm command failed",
			zap.Error(err),
			zap.String("stdout", stdout.String()),
			zap.String("stderr", stderr.String()))
		return fmt.Errorf("helm command failed: %w\nstderr: %s", err, stderr.String())
	}

	if stdout.Len() > 0 {
		e.logger.Info("helm output", zap.String("output", stdout.String()))
	}

	return nil
}

// LoadValuesFile loads and merges a values file
func LoadValuesFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read values file: %w", err)
	}

	var values map[string]interface{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("failed to parse values file: %w", err)
	}

	return values, nil
}

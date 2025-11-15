package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oleksiyp/helmfire/internal/version"
	"github.com/oleksiyp/helmfire/pkg/drift"
	"github.com/oleksiyp/helmfire/pkg/helmstate"
	"github.com/oleksiyp/helmfire/pkg/substitute"
	"github.com/oleksiyp/helmfire/pkg/sync"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	globalLogger     *zap.Logger
	globalSubstitutor *substitute.Manager
)

func main() {
	// Initialize logger
	var err error
	globalLogger, err = zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer globalLogger.Sync()

	// Initialize substitutor
	globalSubstitutor = substitute.NewManager()

	rootCmd := &cobra.Command{
		Use:   "helmfire",
		Short: "Helmfile sync with watching, live substitution, and drift detection",
		Long: `Helmfire extends helmfile with developer-friendly features:
- Watch mode: auto-sync on file changes
- Chart substitution: replace remote charts with local versions
- Image substitution: override container images dynamically
- Drift detection: monitor cluster state vs. desired state
- Daemon mode: background process with API control`,
		Version: version.Version,
	}

	// Add subcommands
	rootCmd.AddCommand(newSyncCmd())
	rootCmd.AddCommand(newChartCmd())
	rootCmd.AddCommand(newImageCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newRemoveCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newSyncCmd() *cobra.Command {
	var (
		watch           bool
		daemon          bool
		driftDetect     bool
		driftInterval   time.Duration
		driftAutoHeal   bool
		driftWebhook    string
		file            string
		environment     string
		selectors       []string
		namespace       string
		kubeContext     string
		dryRun          bool
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize releases (like helmfile sync)",
		Long: `Execute helmfile sync with optional watching and drift detection.

Examples:
  # Basic sync
  helmfire sync

  # Sync with specific helmfile
  helmfire sync -f helmfile.yaml

  # Dry run
  helmfire sync --dry-run

  # Sync to specific namespace
  helmfire sync --namespace production`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if watch || daemon {
				return fmt.Errorf("watch mode and daemon mode not yet implemented (Phase 2 and 4)")
			}

			// Load helmfile
			globalLogger.Info("loading helmfile", zap.String("file", file))
			manager := helmstate.NewManager(file, environment)
			if err := manager.Load(); err != nil {
				return fmt.Errorf("failed to load helmfile: %w", err)
			}

			// Create executor
			executor := sync.NewExecutor(globalLogger, globalSubstitutor)
			executor.SetDryRun(dryRun)
			if namespace != "" {
				executor.SetNamespace(namespace)
			}
			if kubeContext != "" {
				executor.SetKubeContext(kubeContext)
			}

			// Sync repositories
			repos := manager.GetRepositories()
			if len(repos) > 0 {
				globalLogger.Info("syncing repositories", zap.Int("count", len(repos)))
				if err := executor.SyncRepositories(repos); err != nil {
					return fmt.Errorf("failed to sync repositories: %w", err)
				}
			}

			// Get releases
			releases := manager.GetReleases()
			globalLogger.Info("found releases", zap.Int("count", len(releases)))

			// Sync each release
			for _, release := range releases {
				if !manager.IsReleaseInstalled(release) {
					globalLogger.Info("skipping release (installed: false)", zap.String("name", release.Name))
					continue
				}

				if err := executor.SyncRelease(release); err != nil {
					return fmt.Errorf("failed to sync release %s: %w", release.Name, err)
				}
			}

			globalLogger.Info("sync completed successfully")

			// Start drift detection if enabled
			if driftDetect {
				globalLogger.Info("starting drift detection",
					zap.Duration("interval", driftInterval),
					zap.Bool("autoHeal", driftAutoHeal))

				// Create drift detector
				detector := drift.NewDetector(manager, driftInterval, globalLogger)

				// Add stdout notifier
				detector.AddNotifier(drift.NewStdoutNotifier(globalLogger))

				// Add webhook notifier if configured
				if driftWebhook != "" {
					detector.AddNotifier(drift.NewWebhookNotifier(driftWebhook, globalLogger))
				}

				// Enable auto-heal if requested
				if driftAutoHeal {
					healFunc := func(releaseName string) error {
						// Find the release
						for _, release := range releases {
							if release.Name == releaseName {
								globalLogger.Info("healing release", zap.String("name", releaseName))
								return executor.SyncRelease(release)
							}
						}
						return fmt.Errorf("release not found: %s", releaseName)
					}
					detector.EnableAutoHeal(true, healFunc)
				}

				// Create context with signal handling
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// Handle interrupt signals
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

				// Start detector
				if err := detector.Start(ctx); err != nil {
					return fmt.Errorf("failed to start drift detector: %w", err)
				}

				globalLogger.Info("drift detector running, press Ctrl+C to stop")
				fmt.Println("\n✓ Drift detector running...")
				fmt.Printf("  Interval: %s\n", driftInterval)
				fmt.Printf("  Auto-heal: %v\n", driftAutoHeal)
				if driftWebhook != "" {
					fmt.Printf("  Webhook: %s\n", driftWebhook)
				}
				fmt.Println("\nPress Ctrl+C to stop")

				// Wait for interrupt
				<-sigChan
				globalLogger.Info("received interrupt signal, stopping drift detector")
				fmt.Println("\nStopping drift detector...")

				// Stop detector
				if err := detector.Stop(); err != nil {
					return fmt.Errorf("failed to stop drift detector: %w", err)
				}

				fmt.Println("✓ Drift detector stopped")
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch for file changes and auto-sync (Phase 2)")
	cmd.Flags().BoolVar(&daemon, "daemon", false, "Run as background daemon (Phase 4)")
	cmd.Flags().BoolVar(&driftDetect, "drift-detect", false, "Enable drift detection")
	cmd.Flags().DurationVar(&driftInterval, "drift-interval", 30*time.Second, "Drift detection interval")
	cmd.Flags().BoolVar(&driftAutoHeal, "drift-auto-heal", false, "Automatically heal detected drift")
	cmd.Flags().StringVar(&driftWebhook, "drift-webhook", "", "Webhook URL for drift notifications")
	cmd.Flags().StringVarP(&file, "file", "f", "helmfile.yaml", "Path to helmfile")
	cmd.Flags().StringVarP(&environment, "environment", "e", "", "Environment name")
	cmd.Flags().StringSliceVarP(&selectors, "selector", "l", nil, "Label selectors")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Default namespace")
	cmd.Flags().StringVar(&kubeContext, "kube-context", "", "Kubernetes context")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate sync without making changes")

	return cmd
}

func newChartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chart <original> <local-path>",
		Short: "Substitute a chart with a local version",
		Long: `Replace a remote chart reference with a local chart directory.

The substitution applies to all releases using the original chart.
Run 'helmfire sync' after adding substitutions to apply them.

Examples:
  # Replace bitnami/postgresql with local chart
  helmfire chart bitnami/postgresql ./charts/postgresql

  # Replace with absolute path
  helmfire chart stable/mysql /home/user/charts/mysql`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			original := args[0]
			localPath := args[1]

			if err := globalSubstitutor.AddChartSubstitution(original, localPath); err != nil {
				return fmt.Errorf("failed to add chart substitution: %w", err)
			}

			globalLogger.Info("chart substitution added",
				zap.String("original", original),
				zap.String("local", localPath))

			fmt.Printf("✓ Chart substitution added: %s → %s\n", original, localPath)
			fmt.Println("Run 'helmfire sync' to apply the substitution")

			return nil
		},
	}

	return cmd
}

func newImageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image <original> <replacement>",
		Short: "Substitute a container image",
		Long: `Replace a container image reference across all releases.

The substitution is applied during manifest rendering via post-renderer.
Run 'helmfire sync' after adding substitutions to apply them.

Examples:
  # Replace postgres image
  helmfire image postgres:15 localhost:5000/postgres:dev

  # Replace nginx with custom registry
  helmfire image nginx:1.21 myregistry.io/nginx:custom`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			original := args[0]
			replacement := args[1]

			if err := globalSubstitutor.AddImageSubstitution(original, replacement); err != nil {
				return fmt.Errorf("failed to add image substitution: %w", err)
			}

			globalLogger.Info("image substitution added",
				zap.String("original", original),
				zap.String("replacement", replacement))

			fmt.Printf("✓ Image substitution added: %s → %s\n", original, replacement)
			fmt.Println("Run 'helmfire sync' to apply the substitution")

			return nil
		},
	}

	return cmd
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List active substitutions",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "charts",
		Short: "List chart substitutions",
		RunE: func(cmd *cobra.Command, args []string) error {
			subs := globalSubstitutor.ListChartSubstitutions()
			if len(subs) == 0 {
				fmt.Println("No chart substitutions active")
				return nil
			}

			fmt.Println("Active chart substitutions:")
			for _, sub := range subs {
				fmt.Printf("  %s → %s\n", sub.Original, sub.LocalPath)
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "images",
		Short: "List image substitutions",
		RunE: func(cmd *cobra.Command, args []string) error {
			subs := globalSubstitutor.ListImageSubstitutions()
			if len(subs) == 0 {
				fmt.Println("No image substitutions active")
				return nil
			}

			fmt.Println("Active image substitutions:")
			for _, sub := range subs {
				fmt.Printf("  %s → %s\n", sub.Original, sub.Replacement)
			}
			return nil
		},
	})

	return cmd
}

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove substitutions",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "chart <original>",
		Short: "Remove chart substitution",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			original := args[0]
			if err := globalSubstitutor.RemoveChartSubstitution(original); err != nil {
				return err
			}

			fmt.Printf("✓ Chart substitution removed: %s\n", original)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "image <original>",
		Short: "Remove image substitution",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			original := args[0]
			if err := globalSubstitutor.RemoveImageSubstitution(original); err != nil {
				return err
			}

			fmt.Printf("✓ Image substitution removed: %s\n", original)
			return nil
		},
	})

	return cmd
}

package main

import (
	"fmt"
	"os"

	"github.com/oleksiyp/helmfire/internal/version"
	"github.com/spf13/cobra"
)

func main() {
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
	rootCmd.AddCommand(newDaemonCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newRemoveCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize releases (like helmfile sync)",
		Long: `Execute helmfile sync with optional watching and drift detection.

Examples:
  # Basic sync
  helmfire sync

  # Sync with watching
  helmfire sync --watch

  # Sync with drift detection
  helmfire sync --watch --drift-detect --drift-interval=30s

  # Daemon mode
  helmfire sync --watch --daemon`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement sync command
			return fmt.Errorf("sync command not yet implemented")
		},
	}

	cmd.Flags().BoolP("watch", "w", false, "Watch for file changes and auto-sync")
	cmd.Flags().Bool("daemon", false, "Run as background daemon")
	cmd.Flags().Bool("drift-detect", false, "Enable drift detection")
	cmd.Flags().Duration("drift-interval", 30, "Drift detection interval")
	cmd.Flags().Bool("drift-auto-heal", false, "Automatically heal detected drift")
	cmd.Flags().StringP("file", "f", "helmfile.yaml", "Path to helmfile")
	cmd.Flags().StringP("environment", "e", "", "Environment name")
	cmd.Flags().StringSliceP("selector", "l", nil, "Label selectors")
	cmd.Flags().Int("concurrency", 0, "Concurrent releases (0 = unlimited)")
	cmd.Flags().Bool("interactive", false, "Prompt before each release")

	return cmd
}

func newChartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chart <original> <local-path>",
		Short: "Substitute a chart with a local version",
		Long: `Replace a remote chart reference with a local chart directory.

The substitution applies to all releases using the original chart.
Changes to the local chart trigger automatic re-deployment in watch mode.

Examples:
  # Replace bitnami/postgresql with local chart
  helmfire chart bitnami/postgresql ./charts/postgresql

  # Replace with absolute path
  helmfire chart stable/mysql /home/user/charts/mysql`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement chart substitution
			original := args[0]
			localPath := args[1]
			fmt.Printf("Chart substitution: %s -> %s (not yet implemented)\n", original, localPath)
			return nil
		},
	}

	cmd.Flags().String("daemon-socket", "", "Daemon socket path (if daemon is running)")

	return cmd
}

func newImageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image <original> <replacement>",
		Short: "Substitute a container image",
		Long: `Replace a container image reference across all releases.

The substitution is applied during manifest rendering via post-renderer.
Changes trigger automatic re-deployment in watch mode.

Examples:
  # Replace postgres image
  helmfire image postgres:15 localhost:5000/postgres:dev

  # Replace nginx with custom registry
  helmfire image nginx:1.21 myregistry.io/nginx:custom`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement image substitution
			original := args[0]
			replacement := args[1]
			fmt.Printf("Image substitution: %s -> %s (not yet implemented)\n", original, replacement)
			return nil
		},
	}

	cmd.Flags().String("daemon-socket", "", "Daemon socket path (if daemon is running)")

	return cmd
}

func newDaemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage helmfire daemon",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "Start daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("daemon start not yet implemented")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("daemon stop not yet implemented")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show daemon status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("daemon status not yet implemented")
		},
	})

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
			return fmt.Errorf("list charts not yet implemented")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "images",
		Short: "List image substitutions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("list images not yet implemented")
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
			return fmt.Errorf("remove chart not yet implemented")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "image <original>",
		Short: "Remove image substitution",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("remove image not yet implemented")
		},
	})

	return cmd
}

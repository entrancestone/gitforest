package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

// rootFlag holds the value of the persistent --root flag. It overrides both the
// GITFOREST_ROOT environment variable and the configured root; see
// config.ResolveRoot for the full precedence.
var rootFlag string

func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

var rootCmd = &cobra.Command{
	Use:   "gitforest",
	Short: "Manage a pool of git worktrees for parallel AI agent workflows",
	Long: `Gitforest maintains a pool of reusable, pre-warmed git worktrees
so that multiple AI coding agents can work on the same repo in parallel.`,
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return getRunE(cmd, args)
	},
}

func init() {
	rootCmd.SetVersionTemplate(`{{.Version}}` + "\n")
	rootCmd.PersistentFlags().StringVar(&rootFlag, "root", "",
		"Worktree root directory, overriding GITFOREST_ROOT and config; relative paths (e.g. \".\" for an in-project pool) resolve from the repo root")
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

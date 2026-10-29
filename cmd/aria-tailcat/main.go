package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"ariacompute/tailcat/internal/cat"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	root := &cobra.Command{
		Use:           "aria-tailcat",
		Short:         "Deployment launcher for github.com/tailscale/tailcat on ariacompute hosts",
		Long:          "aria-tailcat is the ariacompute deployment wrapper for tailscale/tailcat (a control-plane-free, WireGuard+DERP point-to-point tunnel). It resolves configuration from YAML/flags/env, builds the `tailcat serve` argument vector, and launches the bundled tailcat binary as a long-running service.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := cat.FlagsFromCmd(cmd)
			if err != nil {
				return err
			}
			return cat.Run(cmd.Context(), opts)
		},
	}

	runCmd := &cobra.Command{
		Use:           "run",
		Short:         "Run the tailcat serve service",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := cat.FlagsFromCmd(cmd)
			if err != nil {
				return err
			}
			return cat.Run(cmd.Context(), opts)
		},
	}

	cat.RegisterFlags(root)
	cat.RegisterFlags(runCmd)
	root.AddCommand(runCmd)

	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

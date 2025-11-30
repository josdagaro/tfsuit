package main

import (
	"github.com/spf13/cobra"

	"github.com/josdagaro/tfsuit/internal/config"
	"github.com/josdagaro/tfsuit/internal/rewrite"
)

var (
	write  bool
	dryRun bool
)

func newFixCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fix [path]",
		Short: "Auto-correct Terraform labels to match naming rules",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) == 1 {
				target = args[0]
			}

			// Si --write se pasó, forzamos dryRun=false
			if write {
				dryRun = false
			} else {
				// modo por defecto es dry-run
				dryRun = true
			}

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}
			opts := rewrite.Options{Write: write, DryRun: dryRun}
			return rewrite.Run(target, cfg, opts)
		},
	}

	cmd.Flags().BoolVar(&write, "write", false, "write changes in-place")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview diff only (default true when --write is not supplied)")
	cmd.Flags().StringVarP(&cfgFile, "config", "c", "tfsuit.hcl", "configuration file (HCL or JSON)")
	return cmd
}

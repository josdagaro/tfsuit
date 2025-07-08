package main

import (
    "github.com/spf13/cobra"

    "github.com/josdagaro/tfsuit/internal/config"
    "github.com/josdagaro/tfsuit/internal/rewrite"
)

var (
    write   bool
    dryRun  bool
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
            cfg, err := config.Load(cfgFile)
            if err != nil {
                return err
            }
            opts := rewrite.Options{Write: write && !dryRun, DryRun: dryRun}
            return rewrite.Run(target, cfg, opts)
        },
    }
    cmd.Flags().BoolVar(&write, "write", false, "write changes in-place (default false)")
    cmd.Flags().BoolVar(&dryRun, "dry-run", true, "show proposed diff instead of writing (default true)")
    return cmd
}

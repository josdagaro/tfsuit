package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/josdagaro/tfsuit/internal/config"
	"github.com/josdagaro/tfsuit/internal/rewrite"
)

var (
	write    bool
	dryRun   bool
	fixTypes string
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

			allowedKinds, err := parseFixTypesFlag(fixTypes)
			if err != nil {
				return err
			}

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}
			opts := rewrite.Options{Write: write, DryRun: dryRun, FixKinds: allowedKinds}
			return rewrite.Run(target, cfg, opts)
		},
	}

	cmd.Flags().BoolVar(&write, "write", false, "write changes in-place")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview diff only (default true when --write is not supplied)")
	cmd.Flags().StringVarP(&cfgFile, "config", "c", "tfsuit.hcl", "configuration file (HCL or JSON)")
	cmd.Flags().StringVar(&fixTypes, "fix-types", "", "comma-separated kinds to fix (file,variable,output,module,data,resource)")
	return cmd
}

func parseFixTypesFlag(flag string) (map[string]bool, error) {
	if flag == "" {
		return nil, nil
	}
	valid := map[string]struct{}{
		"file":     {},
		"variable": {},
		"output":   {},
		"module":   {},
		"data":     {},
		"resource": {},
		"spacing":  {},
	}
	kinds := map[string]bool{}
	for _, part := range strings.Split(flag, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := valid[part]; !ok {
			return nil, fmt.Errorf("unknown fix type %q (valid: file,variable,output,module,data,resource,spacing)", part)
		}
		kinds[part] = true
	}
	if len(kinds) == 0 {
		return nil, fmt.Errorf("fix-types flag requires at least one type")
	}
	return kinds, nil
}

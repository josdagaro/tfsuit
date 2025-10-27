package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/josdagaro/tfsuit/internal/config"
	"github.com/josdagaro/tfsuit/internal/engine"
)

// ← Goreleaser sobreescribe esto con -ldflags "-X main.version={{ .Version }}"
var version = "dev"

var (
	cfgFile string
	format  string
	fail    bool
)

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tfsuit [path]",
		Short: "Fast, opinionated Terraform naming linter",
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

			findings, err := engine.Scan(target, cfg)
			if err != nil {
				return err
			}
			output := engine.Format(findings, format)
			fmt.Print(output)
			if fail && len(findings) > 0 {
				return fmt.Errorf("%d naming violations", len(findings))
			}
			return nil
		},
	}

	// ---- versión (solo número) ----
	cmd.Version = version
	cmd.SetVersionTemplate("{{.Version}}\n") // imprime solo la versión

	cmd.Flags().StringVarP(&cfgFile, "config", "c", "tfsuit.hcl", "configuration file (HCL or JSON)")
	cmd.Flags().StringVarP(&format, "format", "f", "pretty", "output format: pretty|json|sarif")
	cmd.Flags().BoolVar(&fail, "fail", false, "return non-zero exit if violations found")
	cmd.AddCommand(newFixCmd())

	return cmd
}

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

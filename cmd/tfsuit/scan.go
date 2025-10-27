package main

import "github.com/spf13/cobra"

func newScanCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "scan [path]",
		Short: "Scan for Terraform naming violations",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) == 1 {
				target = args[0]
			}
			return runScan(target)
		},
	}

	// mismos flags que el comando raíz
	c.Flags().StringVarP(&cfgFile, "config", "c", "tfsuit.hcl", "configuration file (HCL or JSON)")
	c.Flags().StringVarP(&format, "format", "f", "pretty", "output format: pretty|json|sarif")
	c.Flags().BoolVar(&fail, "fail", false, "return non-zero exit if violations found")

	return c
}

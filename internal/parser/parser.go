package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	hcl "github.com/hashicorp/hcl/v2"
	hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/josdagaro/tfsuit/internal/config"
	"github.com/josdagaro/tfsuit/internal/model"
)

// Discover returns .tf files recursively
func Discover(root string) ([]string, error) {
	var list []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".tf" {
			list = append(list, path)
		}
		return nil
	})
	return list, err
}

// ParseFile extracts identifiers for variables, outputs, modules and resources,
// evaluates them against naming rules, and returns violations.
func ParseFile(path string, cfg *config.Config) ([]model.Finding, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	file, diags := hclsyntax.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("%s: %s", path, diags.Error())
	}

	syntaxBody, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("unexpected body type in %s", path)
	}

	var findings []model.Finding

	for _, block := range syntaxBody.Blocks {
		switch block.Type {
		case "variable":
			if len(block.Labels) == 0 {
				continue
			}
			name := block.Labels[0]
			evalRule(&findings, path, block, "variable", name, &cfg.Variables)

		case "output":
			if len(block.Labels) == 0 {
				continue
			}
			name := block.Labels[0]
			evalRule(&findings, path, block, "output", name, &cfg.Outputs)

		case "module":
			if len(block.Labels) == 0 {
				continue
			}
			name := block.Labels[0]
			evalRule(&findings, path, block, "module", name, &cfg.Modules)

		case "resource":
			// resource blocks have two labels: TYPE and NAME
			if len(block.Labels) < 2 {
				continue
			}
			name := block.Labels[1]
			evalRule(&findings, path, block, "resource", name, &cfg.Resources)
		}
	}

	return findings, nil
}

// evalRule checks a single identifier against its rule and appends a finding if needed.
func evalRule(findings *[]model.Finding, path string, block *hclsyntax.Block, kind, name string, rule *config.Rule) {
	if rule.IsIgnored(name) {
		return
	}
	if rule.Matches(name) {
		return
	}
	*findings = append(*findings, model.Finding{
		File:    path,
		Line:    block.DefRange().Start.Line,
		Kind:    kind,
		Name:    name,
		Message: fmt.Sprintf("%s '%s' does not match pattern %s", kind, name, rule.Pattern),
	})
}

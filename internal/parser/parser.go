package parser

import (
	"fmt"
	"os"

	hcl "github.com/hashicorp/hcl/v2"
	hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/josdagaro/tfsuit/internal/config"
	"github.com/josdagaro/tfsuit/internal/model"
)

// ParseFile extrae identificadores y devuelve violaciones según las reglas.
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
			// resource tiene dos labels: TYPE y NAME
			if len(block.Labels) < 2 {
				continue
			}
			name := block.Labels[1]
			evalRule(&findings, path, block, "resource", name, &cfg.Resources)

		case "data":
			if len(block.Labels) < 2 {
				continue
			}
			name := block.Labels[1]
			evalRule(&findings, path, block, "data", name, cfg.Data)
		}
	}

	return findings, nil
}

// evalRule evalúa un identificador contra su regla y añade un finding si aplica.
func evalRule(findings *[]model.Finding, path string, block *hclsyntax.Block, kind, name string, rule *config.Rule) {
	if rule == nil {
		return
	}
	if rule.IsIgnored(name) {
		return
	}

	if rule.RequiresProvider() {
		if !hasRequiredProvider(block, kind) {
			*findings = append(*findings, model.Finding{
				File:    path,
				Line:    block.DefRange().Start.Line,
				Kind:    kind,
				Name:    name,
				Message: providerMessage(kind, name),
			})
		}
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

func hasRequiredProvider(block *hclsyntax.Block, kind string) bool {
	switch kind {
	case "module":
		attr, ok := block.Body.Attributes["providers"]
		if !ok {
			return false
		}
		if obj, ok := attr.Expr.(*hclsyntax.ObjectConsExpr); ok {
			return len(obj.Items) > 0
		}
		return true

	case "resource", "data":
		_, ok := block.Body.Attributes["provider"]
		return ok
	}
	return true
}

func providerMessage(kind, name string) string {
	if kind == "module" {
		return fmt.Sprintf("%s '%s' must declare at least one providers mapping", kind, name)
	}
	return fmt.Sprintf("%s '%s' must set a provider", kind, name)
}

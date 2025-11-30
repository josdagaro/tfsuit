package parser

import (
	"fmt"
	"os"
	"strings"

	hcl "github.com/hashicorp/hcl/v2"
	hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/josdagaro/tfsuit/internal/config"
	"github.com/josdagaro/tfsuit/internal/model"
)

type blockInfo struct {
	Kind       string
	Name       string
	StartLine  int
	EndLine    int
	SingleLine bool
}

func newBlockInfo(kind, name string, block *hclsyntax.Block) blockInfo {
	rng := block.Range()
	return blockInfo{
		Kind:       kind,
		Name:       name,
		StartLine:  rng.Start.Line,
		EndLine:    rng.End.Line,
		SingleLine: rng.Start.Line == rng.End.Line,
	}
}

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
	var blockInfos []blockInfo

	for _, block := range syntaxBody.Blocks {
		switch block.Type {
		case "variable":
			if len(block.Labels) == 0 {
				continue
			}
			name := block.Labels[0]
			evalRule(&findings, path, block, "variable", name, &cfg.Variables)
			blockInfos = append(blockInfos, newBlockInfo("variable", name, block))

		case "output":
			if len(block.Labels) == 0 {
				continue
			}
			name := block.Labels[0]
			evalRule(&findings, path, block, "output", name, &cfg.Outputs)
			blockInfos = append(blockInfos, newBlockInfo("output", name, block))

		case "module":
			if len(block.Labels) == 0 {
				continue
			}
			name := block.Labels[0]
			evalRule(&findings, path, block, "module", name, &cfg.Modules)
			blockInfos = append(blockInfos, newBlockInfo("module", name, block))

		case "resource":
			// resource tiene dos labels: TYPE y NAME
			if len(block.Labels) < 2 {
				continue
			}
			name := block.Labels[1]
			evalRule(&findings, path, block, "resource", name, &cfg.Resources)
			blockInfos = append(blockInfos, newBlockInfo("resource", name, block))

		case "data":
			if len(block.Labels) < 2 {
				continue
			}
			name := block.Labels[1]
			evalRule(&findings, path, block, "data", name, cfg.Data)
			blockInfos = append(blockInfos, newBlockInfo("data", name, block))
		}
	}

	if cfg.Spacing != nil && cfg.Spacing.EnabledValue() {
		spacingFindings := checkBlockSpacing(path, src, blockInfos, cfg.Spacing)
		findings = append(findings, spacingFindings...)
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

func checkBlockSpacing(path string, src []byte, infos []blockInfo, spacing *config.BlockSpacing) []model.Finding {
	if spacing == nil || !spacing.EnabledValue() || len(infos) < 2 {
		return nil
	}
	lines := splitLines(src)
	var findings []model.Finding

	for i := 0; i < len(infos)-1; i++ {
		current := infos[i]
		next := infos[i+1]

		if spacing.AllowCompactKind(current.Kind) && spacing.AllowCompactKind(next.Kind) &&
			current.SingleLine && next.SingleLine {
			continue
		}

		actual := countBlankLinesBetween(lines, current.EndLine, next.StartLine)
		if actual >= spacing.MinLines() {
			continue
		}
		f := model.Finding{
			File: path,
			Line: next.StartLine,
			Kind: "spacing",
			Name: fmt.Sprintf("%s/%s", current.Kind, next.Kind),
			Message: fmt.Sprintf(
				"expected at least %d blank line(s) between %s '%s' and %s '%s'",
				spacing.MinLines(), current.Kind, current.Name, next.Kind, next.Name,
			),
		}
		findings = append(findings, f)
	}
	return findings
}

func splitLines(src []byte) []string {
	var lines []string
	start := 0
	for i, b := range src {
		if b == '\n' {
			lines = append(lines, string(src[start:i]))
			start = i + 1
		}
	}
	if start <= len(src) {
		lines = append(lines, string(src[start:]))
	}
	return lines
}

func countBlankLinesBetween(lines []string, endLine, nextStart int) int {
	if nextStart <= endLine+1 {
		return 0
	}
	blank := 0
	for lineNum := endLine + 1; lineNum <= nextStart-1; lineNum++ {
		idx := lineNum - 1
		if idx < 0 || idx >= len(lines) {
			continue
		}
		if strings.TrimSpace(lines[idx]) == "" {
			blank++
		}
	}
	return blank
}

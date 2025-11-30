package rewrite

import (
	"os"
	"path/filepath"
	"testing"

	hcl "github.com/hashicorp/hcl/v2"
	hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"
)

func parseBlock(t *testing.T, content string) *hclsyntax.Block {
	t.Helper()
	file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("parse config: %s", diags.Error())
	}
	body := file.Body.(*hclsyntax.Body)
	if len(body.Blocks) == 0 {
		t.Fatalf("no block parsed")
	}
	return body.Blocks[0]
}

func TestNeedsProviderAssignmentAndType(t *testing.T) {
	mod := parseBlock(t, `module "m" { source = "./child" }`)
	if !needsProviderAssignment(mod, "module") {
		t.Fatalf("module without providers should need assignment")
	}
	modWith := parseBlock(t, `module "m" { providers = { aws = aws.primary } }`)
	if needsProviderAssignment(modWith, "module") {
		t.Fatalf("module with providers map should not need assignment")
	}

	res := parseBlock(t, `resource "aws_s3_bucket" "logs" { provider = aws.primary }`)
	if needsProviderAssignment(res, "resource") {
		t.Fatalf("resource with provider should be satisfied")
	}
	if providerTypeFromBlock(res) != "aws" {
		t.Fatalf("provider type not inferred from block labels")
	}
}

func TestAliasHelpers(t *testing.T) {
	if got := providerTypeFromAlias("aws.primary"); got != "aws" {
		t.Fatalf("providerTypeFromAlias returned %s", got)
	}
	if got := providerTypeFromAlias("aws"); got != "aws" {
		t.Fatalf("providerTypeFromAlias without alias returned %s", got)
	}

	expr, diags := hclsyntax.ParseExpression([]byte(`[aws.primary, "aws.secondary"]`), "expr.hcl", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("parse expression: %s", diags.Error())
	}
	list := aliasListFromExpr(expr)
	if len(list) != 2 || list[0] != "aws.primary" || list[1] != "aws.secondary" {
		t.Fatalf("aliasListFromExpr mismatch: %v", list)
	}

	strExpr, diags := hclsyntax.ParseExpression([]byte(`"aws.primary"`), "expr.hcl", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("parse expression string: %s", diags.Error())
	}
	if list := aliasListFromExpr(strExpr); len(list) != 1 {
		t.Fatalf("aliasList from string unexpected: %v", list)
	}

	tmpl := &hclsyntax.TemplateExpr{
		Parts: []hclsyntax.Expression{
			&hclsyntax.TemplateWrapExpr{
				Wrapped: &hclsyntax.ScopeTraversalExpr{
					Traversal: hcl.Traversal{
						hcl.TraverseRoot{Name: "aws"},
						hcl.TraverseAttr{Name: "primary"},
					},
				},
			},
		},
	}
	if got := objectKeyToString(tmpl); got != "aws.primary" {
		t.Fatalf("objectKeyToString template mismatch: %s", got)
	}
}

func TestModuleSourceHelpers(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	block := parseBlock(t, `module "mod" { source = "./child" }`)
	source, ok := moduleSourceString(block)
	if !ok || source != "./child" {
		t.Fatalf("moduleSourceString failed: %s %v", source, ok)
	}
	resolved, ok := resolveModuleSource(dir, dir, source)
	if !ok || resolved != child {
		t.Fatalf("resolveModuleSource failed: %s %v", resolved, ok)
	}
}

func TestProviderScopeAliasManagement(t *testing.T) {
	scope := newProviderScope("root")
	alias := scope.addAlias("aws.primary", false)
	if alias == nil || alias.Defined {
		t.Fatalf("expected alias defined=false initially")
	}
	alias = scope.addAlias("aws.primary", true)
	if !alias.Defined {
		t.Fatalf("expected alias to become defined")
	}
	scope.allowAlias("aws.primary")
	if len(scope.Allowed) != 1 {
		t.Fatalf("allowAlias should populate Allowed map")
	}
	if list := scope.allowedAliasList(); len(list) != 1 {
		t.Fatalf("allowedAliasList mismatch: %v", list)
	}
	if name := buildProviderAliasName("aws", "foo"); name != "aws.foo" {
		t.Fatalf("buildProviderAliasName wrong: %s", name)
	}
	if name := buildProviderAliasName("aws", ""); name != "aws" {
		t.Fatalf("buildProviderAliasName default wrong: %s", name)
	}
}

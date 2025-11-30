package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/josdagaro/tfsuit/internal/config"
	"github.com/josdagaro/tfsuit/internal/parser"
)

func TestParserDetectsViolations(t *testing.T) {
	root := filepath.FromSlash("../../samples/simple")
	cfg, err := config.Load(filepath.Join(root, "tfsuit.hcl"))
	if err != nil {
		t.Fatalf("load cfg: %v", err)
	}

	badFile := filepath.Join(root, "bad.tf")
	findings, err := parser.ParseFile(badFile, cfg)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(findings) != 3 {
		t.Fatalf("want 3 findings, got %d", len(findings))
	}

	goodFile := filepath.Join(root, "good.tf")
	findings, err = parser.ParseFile(goodFile, cfg)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("want 0 findings, got %d", len(findings))
	}
}

func BenchmarkParse100(b *testing.B) {
	root := filepath.FromSlash("../../samples/simple")
	cfg, _ := config.Load(filepath.Join(root, "tfsuit.hcl"))
	file := filepath.Join(root, "good.tf")
	for i := 0; i < b.N; i++ {
		parser.ParseFile(file, cfg)
	}
}

func TestSpacingFindings(t *testing.T) {
	dir := t.TempDir()
	tfPath := filepath.Join(dir, "main.tf")
	content := `
module "a" {}
module "b" {}
`
	if err := os.WriteFile(tfPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write tf: %v", err)
	}
	cfgPath := filepath.Join(dir, "tfsuit.hcl")
	cfgContent := `
variables { pattern = ".*" }
outputs   { pattern = ".*" }
modules   { pattern = ".*" }
resources { pattern = ".*" }
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load cfg: %v", err)
	}
	findings, err := parser.ParseFile(tfPath, cfg)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.Kind == "spacing" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected spacing finding, got %v", findings)
	}
}

func TestRequireProviderEnforcement(t *testing.T) {
	dir := t.TempDir()

	cfgContent := `
variables {
  pattern = ".*"
}

outputs {
  pattern = ".*"
}

modules {
  pattern = ".*"
}

resources {
  pattern = ".*"
  require_provider = true
}

data {
  pattern = ".*"
  require_provider = true
}
`
	cfgPath := filepath.Join(dir, "tfsuit.hcl")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	tfContent := `
module "no_providers" {
  source = "../"
}

module "with_providers" {
  source = "../"
  providers = {
    aws = aws.primary
  }
}

resource "aws_s3_bucket" "no_provider" {}

resource "aws_s3_bucket" "with_provider" {
  provider = aws.primary
}

data "aws_region" "missing_data" {}

data "aws_region" "with_provider" {
  provider = aws.primary
}
`
	tfPath := filepath.Join(dir, "main.tf")
	if err := os.WriteFile(tfPath, []byte(tfContent), 0o644); err != nil {
		t.Fatalf("write tf: %v", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load cfg: %v", err)
	}

	findings, err := parser.ParseFile(tfPath, cfg)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(findings) != 3 {
		t.Fatalf("expected 3 provider findings, got %d", len(findings))
	}

	wantMsgs := []string{
		"module 'no_providers' must declare at least one providers mapping",
		"resource 'no_provider' must set a provider",
		"data 'missing_data' must set a provider",
	}
	for _, want := range wantMsgs {
		found := false
		for _, f := range findings {
			if f.Message == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing finding %q", want)
		}
	}
}

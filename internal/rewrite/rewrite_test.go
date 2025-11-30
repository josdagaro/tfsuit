package rewrite_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/josdagaro/tfsuit/internal/config"
	"github.com/josdagaro/tfsuit/internal/parser"
	"github.com/josdagaro/tfsuit/internal/rewrite"
)

func TestFixWritesFiles(t *testing.T) {
	tmp := t.TempDir()

	// copia el fixture simple al tmpdir
	src := filepath.FromSlash("../../samples/simple")
	if err := copyDir(src, tmp); err != nil {
		t.Fatalf("copyDir: %v", err)
	}

	cfg, err := config.Load(filepath.Join(tmp, "tfsuit.hcl"))
	if err != nil {
		t.Fatalf("load cfg: %v", err)
	}

	if err := rewrite.Run(tmp, cfg, rewrite.Options{Write: true}); err != nil {
		t.Fatalf("fix: %v", err)
	}

	// volver a parsear bad.tf: ya no debe haber violaciones
	bad := filepath.Join(tmp, "bad.tf")
	findings, err := parser.ParseFile(bad, cfg)
	if err != nil {
		t.Fatalf("parse after fix: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings after fix, got %d", len(findings))
	}
}

func TestFixAddsProviders(t *testing.T) {
	tmp := t.TempDir()

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
	if err := os.WriteFile(filepath.Join(tmp, "tfsuit.hcl"), []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	tfContent := `
provider "aws" {
  alias  = "primary"
  region = "us-east-1"
}

module "demo" {
  source = "../.."
}

resource "aws_s3_bucket" "logs" {
  bucket = "test"
}

data "aws_region" "current" {}
`
	mainTf := filepath.Join(tmp, "main.tf")
	if err := os.WriteFile(mainTf, []byte(tfContent), 0o644); err != nil {
		t.Fatalf("write tf: %v", err)
	}

	cfg, err := config.Load(filepath.Join(tmp, "tfsuit.hcl"))
	if err != nil {
		t.Fatalf("load cfg: %v", err)
	}

	if err := rewrite.Run(tmp, cfg, rewrite.Options{Write: true}); err != nil {
		t.Fatalf("fix with providers: %v", err)
	}

	out, err := os.ReadFile(mainTf)
	if err != nil {
		t.Fatalf("read tf: %v", err)
	}

	if !strings.Contains(string(out), `provider = aws.primary`) {
		t.Fatalf("resource missing provider assignment:\n%s", out)
	}
	if !strings.Contains(string(out), `data "aws_region" "current" {
  provider = aws.primary`) {
		t.Fatalf("data missing provider assignment:\n%s", out)
	}
	if !strings.Contains(string(out), `providers = {
    aws = aws.primary
  }`) {
		t.Fatalf("module missing providers mapping:\n%s", out)
	}
}

func TestFixFailsWithoutProvidersCreatesScaffold(t *testing.T) {
	tmp := t.TempDir()

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
`
	if err := os.WriteFile(filepath.Join(tmp, "tfsuit.hcl"), []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	tfContent := `
module "demo" {
  source = "../.."
}
`
	mainTf := filepath.Join(tmp, "main.tf")
	if err := os.WriteFile(mainTf, []byte(tfContent), 0o644); err != nil {
		t.Fatalf("write tf: %v", err)
	}

	cfg, err := config.Load(filepath.Join(tmp, "tfsuit.hcl"))
	if err != nil {
		t.Fatalf("load cfg: %v", err)
	}

	err = rewrite.Run(tmp, cfg, rewrite.Options{Write: true})
	if err == nil {
		t.Fatalf("expected error when no providers defined")
	}

	providersFile := filepath.Join(tmp, "providers.tf")
	info, statErr := os.Stat(providersFile)
	if statErr != nil {
		t.Fatalf("providers scaffold not created: %v", statErr)
	}
	if info.Size() == 0 {
		t.Fatalf("providers scaffold is empty")
	}
}

func TestFixPropagatesProvidersThroughModules(t *testing.T) {
	tmp := t.TempDir()

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
`
	writeFile(t, filepath.Join(tmp, "tfsuit.hcl"), cfgContent)

	providersTf := `
provider "aws" {
  alias  = "virginia"
  region = "us-east-1"
}

provider "aws" {
  alias  = "ohio"
  region = "us-east-2"
}
`
	writeFile(t, filepath.Join(tmp, "providers.tf"), providersTf)

	mainTf := `
module "backend" {
  source = "./modules/backend"
}
`
	writeFile(t, filepath.Join(tmp, "main.tf"), mainTf)

	moduleTf := `
terraform {
  required_providers {
    aws = {
      configuration_aliases = [aws.virginia, aws.ohio]
    }
  }
}

module "ecs" {
  source = "./ecs"
}

resource "aws_s3_bucket" "logs" {}
`
	writeFile(t, filepath.Join(tmp, "modules/backend/main.tf"), moduleTf)

	subModuleTf := `
terraform {
  required_providers {
    aws = {
      configuration_aliases = [aws.ohio]
    }
  }
}

resource "aws_iam_role" "app" {}
`
	writeFile(t, filepath.Join(tmp, "modules/backend/ecs/main.tf"), subModuleTf)

	cfg, err := config.Load(filepath.Join(tmp, "tfsuit.hcl"))
	if err != nil {
		t.Fatalf("load cfg: %v", err)
	}

	if err := rewrite.Run(tmp, cfg, rewrite.Options{Write: true}); err != nil {
		t.Fatalf("fix nested modules: %v", err)
	}

	rootMain, err := os.ReadFile(filepath.Join(tmp, "main.tf"))
	if err != nil {
		t.Fatalf("read root main: %v", err)
	}
	if !strings.Contains(string(rootMain), `aws.virginia = aws.virginia`) {
		t.Fatalf("root providers missing virginia mapping:\n%s", rootMain)
	}
	if !strings.Contains(string(rootMain), `aws.ohio = aws.ohio`) {
		t.Fatalf("root providers missing ohio mapping:\n%s", rootMain)
	}

	outBackend, err := os.ReadFile(filepath.Join(tmp, "modules/backend/main.tf"))
	if err != nil {
		t.Fatalf("read backend: %v", err)
	}
	if !strings.Contains(string(outBackend), `provider = aws.ohio`) {
		t.Fatalf("backend resource missing provider:\n%s", outBackend)
	}

	outSub, err := os.ReadFile(filepath.Join(tmp, "modules/backend/ecs/main.tf"))
	if err != nil {
		t.Fatalf("read submodule: %v", err)
	}
	if !strings.Contains(string(outSub), `provider = aws.ohio`) {
		t.Fatalf("submodule resource missing provider:\n%s", outSub)
	}
}

func TestScanFileAfterFix(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", "samples", "simple", "tfsuit.hcl"))
	if err != nil {
		t.Fatalf("load cfg: %v", err)
	}
	path := filepath.Join("..", "..", "samples", "simple", "bad.tf")
	findings, err := rewrite.ScanFileAfterFix(path, cfg)
	if err != nil {
		t.Fatalf("ScanFileAfterFix: %v", err)
	}
	if len(findings) == 0 {
		t.Fatalf("expected findings from sample bad.tf")
	}
}

// utilidades -----------------------------------------------------------------

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, _ := os.ReadFile(p)
		return os.WriteFile(target, b, info.Mode())
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

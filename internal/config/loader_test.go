package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestLoadHCLSetsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "tfsuit.hcl", `
variables {
  pattern = "^[a-z]+$"
  ignore_exact = ["foo"]
}

outputs {
  pattern = ".*"
}

modules {
  pattern = "^[a-z]+$"
  require_provider = false
}

resources {
  pattern = "^[a-z]+$"
}

files {
  pattern = "^[a-z0-9_]+\\.tf$"
}
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Variables.IsIgnored("foo") {
		t.Fatalf("ignore_exact not respected")
	}
	if cfg.Modules.RequiresProvider() {
		t.Fatalf("module require_provider override not applied")
	}
	if !cfg.Resources.Matches("abc") {
		t.Fatalf("resource rule not compiled")
	}
	assertDefaultRule := func(rule *Rule, name string) {
		if rule == nil || rule.Pattern != ".*" {
			t.Fatalf("%s rule should be defaulted", name)
		}
		if rule.RequiresProvider() {
			t.Fatalf("%s default require_provider should be false", name)
		}
	}

	assertDefaultRule(cfg.Data, "data")
	if cfg.Files == nil || cfg.Files.Pattern != `^[a-z0-9_]+\.tf$` {
		t.Fatalf("files pattern not loaded")
	}
	if cfg.Files.IsIgnored("foo.tf") {
		t.Fatalf("files ignore default unexpected")
	}
}

func TestLoadJSON(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "tfsuit.json", `{
  "variables": { "pattern": ".*" },
  "outputs":   { "pattern": ".*" },
  "modules":   { "pattern": ".*" },
  "resources": { "pattern": ".*", "require_provider": true },
  "data":      { "pattern": ".*", "require_provider": true }
}`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load json: %v", err)
	}
	if !cfg.Resources.RequiresProvider() || !cfg.Data.RequiresProvider() {
		t.Fatalf("require_provider true not honored in JSON")
	}
}

func TestRuleCompileErrors(t *testing.T) {
	r := Rule{Pattern: "["}
	if err := r.compile(); err == nil {
		t.Fatalf("expected compile error for invalid regexp")
	}

	r = Rule{Pattern: ".*", IgnoreRegex: []string{"["}}
	if err := r.compile(); err == nil {
		t.Fatalf("expected ignore_regex compile error")
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.hcl"))
	if err == nil {
		t.Fatalf("expected error for missing file")
	}
}

func TestRuleIgnoreRegex(t *testing.T) {
	r := Rule{Pattern: ".*", IgnoreRegex: []string{"^tmp"}}
	if err := r.compile(); err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !r.IsIgnored("tmp1") {
		t.Fatalf("regex ignore failed")
	}
}

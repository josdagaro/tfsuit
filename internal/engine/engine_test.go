package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/josdagaro/tfsuit/internal/config"
	"github.com/josdagaro/tfsuit/internal/model"
)

func samplePath(parts ...string) string {
	all := append([]string{"..", ".."}, parts...)
	return filepath.Join(all...)
}

func TestScanSamplesSimple(t *testing.T) {
	cfg, err := config.Load(samplePath("samples", "simple", "tfsuit.hcl"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	findings, stats, err := Scan(samplePath("samples", "simple"), cfg)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if stats.Files == 0 {
		t.Fatalf("expected files > 0")
	}
	if len(findings) == 0 {
		t.Fatalf("expected findings in sample")
	}
}

func TestFormatModes(t *testing.T) {
	findings := []model.Finding{{
		File:    "main.tf",
		Line:    1,
		Kind:    "module",
		Name:    "bad",
		Message: "broken",
	}}
	stats := &ScanStats{Files: 1, Duration: time.Second}

	if out := Format(findings, "json", stats); !strings.Contains(out, `"file"`) {
		t.Fatalf("json output missing fields: %s", out)
	}
	if out := Format(findings, "sarif", stats); !strings.Contains(out, `"results"`) {
		t.Fatalf("sarif output malformed: %s", out)
	}
	pretty := Format(findings, "pretty", stats)
	if !strings.Contains(pretty, "Violations:") {
		t.Fatalf("pretty output missing violations: %s", pretty)
	}
	empty := Format(nil, "pretty", stats)
	if !strings.Contains(empty, "No naming violations") {
		t.Fatalf("pretty empty message missing: %s", empty)
	}
}

func TestScanReportsFilePatternViolations(t *testing.T) {
	dir := t.TempDir()
	tfPath := filepath.Join(dir, "Bad-Name.tf")
	if err := os.WriteFile(tfPath, []byte(`resource "aws_s3_bucket" "logs" {}`), 0o644); err != nil {
		t.Fatalf("write tf: %v", err)
	}

	cfgContent := `
files { pattern = "^[a-z0-9_]+\\.tf$" }
variables { pattern = ".*" }
outputs   { pattern = ".*" }
modules   { pattern = ".*" }
resources { pattern = ".*" }
`
	cfgPath := filepath.Join(dir, "tfsuit.hcl")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load cfg: %v", err)
	}

	findings, _, err := Scan(dir, cfg)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.Kind == "file" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected file naming violation, got %v", findings)
	}
}

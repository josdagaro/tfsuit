package engine

import (
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

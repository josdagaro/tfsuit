package parser_test

import (
	"path/filepath"
	"testing"

	"github.com/josdagaro/tfsuit/internal/config"
	"github.com/josdagaro/tfsuit/internal/parser"
)

func TestParserDetectsViolations(t *testing.T) {
	root := filepath.FromSlash("../testdata/simple")
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
	root := filepath.FromSlash("../../testdata/simple")
	cfg, _ := config.Load(filepath.Join(root, "tfsuit.hcl"))
	file := filepath.Join(root, "good.tf")
	for i := 0; i < b.N; i++ {
		parser.ParseFile(file, cfg)
	}
}

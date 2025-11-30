package parser

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestDiscoverSkipsTerraformDir(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "main.tf")
	if err := os.WriteFile(valid, []byte("resource \"aws_s3_bucket\" \"a\" {}"), 0o644); err != nil {
		t.Fatalf("write valid: %v", err)
	}
	upper := filepath.Join(dir, "UPPER.TF")
	if err := os.WriteFile(upper, []byte(""), 0o644); err != nil {
		t.Fatalf("write upper: %v", err)
	}
	hiddenDir := filepath.Join(dir, ".terraform")
	if err := os.Mkdir(hiddenDir, 0o755); err != nil {
		t.Fatalf("mkdir hidden: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "ignored.tf"), []byte(""), 0o644); err != nil {
		t.Fatalf("write ignored: %v", err)
	}

	files, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	sort.Strings(files)
	expected := []string{upper, valid}
	sort.Strings(expected)
	sort.Strings(files)
	if len(files) != 2 || files[0] != expected[0] || files[1] != expected[1] {
		t.Fatalf("expected %v, got %v", expected, files)
	}
}

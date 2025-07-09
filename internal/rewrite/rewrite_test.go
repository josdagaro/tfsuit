package rewrite_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/josdagaro/tfsuit/internal/config"
	"github.com/josdagaro/tfsuit/internal/parser"
	"github.com/josdagaro/tfsuit/internal/rewrite"
)

func TestFixWritesFiles(t *testing.T) {
	tmp := t.TempDir()

	// copia el fixture simple al tmpdir
	src := filepath.FromSlash("../testdata/simple") // ← ubicación real
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

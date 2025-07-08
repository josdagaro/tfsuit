package rewrite_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/josdagaro/tfsuit/internal/config"
	"github.com/josdagaro/tfsuit/internal/rewrite"
)

func TestFixWritesFiles(t *testing.T) {
	// copiamos el fixture a un tmpdir porque vamos a escribir
	tmp := t.TempDir()
	src := filepath.FromSlash("../../testdata/simple")
	if err := copyDir(src, tmp); err != nil {
		t.Fatalf("copy: %v", err)
	}

	cfg, _ := config.Load(filepath.Join(tmp, "tfsuit.hcl"))
	if err := rewrite.Run(tmp, cfg, rewrite.Options{Write: true}); err != nil {
		t.Fatalf("fix: %v", err)
	}

	// volvemos a escanear: no debe quedar ninguna violación
	bad := filepath.Join(tmp, "bad.tf")
	findings, _ := rewrite.ScanFileAfterFix(bad, cfg) // helper opcional
	if len(findings) != 0 {
		t.Fatalf("still have violations after fix")
	}
}

// utilidades -----------------------------------------------------------------

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil { return err }
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if info.IsDir() { return os.MkdirAll(target, 0o755) }
		b, _ := os.ReadFile(p)
		return os.WriteFile(target, b, info.Mode())
	})
}

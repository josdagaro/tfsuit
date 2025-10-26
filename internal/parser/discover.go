package parser

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// Discover devuelve la lista de archivos .tf recorriendo recursivamente.
// Se saltan algunos directorios comunes (.git, .terraform, vendor).
func Discover(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			switch d.Name() {
			case ".git", ".terraform", "vendor":
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(strings.ToLower(path), ".tf") {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

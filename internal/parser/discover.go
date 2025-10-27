package parser

import (
	"io/fs"
	"path/filepath"
)

// Discover devuelve todos los .tf recursivamente (ignora .terraform/)
func Discover(root string) ([]string, error) {
	var list []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// ignora los directorios de proveedor
		if d.IsDir() && d.Name() == ".terraform" {
			return filepath.SkipDir
		}
		if !d.IsDir() && filepath.Ext(path) == ".tf" {
			list = append(list, path)
		}
		return nil
	})
	return list, err
}

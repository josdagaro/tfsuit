package parser

import (
    "fmt"
    "io/fs"
    "path/filepath"

    hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"
)

type identifier struct {
    file string
    line int
    kind string
    name string
}

// Discover returns .tf files recursively
func Discover(root string) ([]string, error) {
    var list []string
    err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if !d.IsDir() && filepath.Ext(path) == ".tf" {
            list = append(list, path)
        }
        return nil
    })
    return list, err
}

// ParseFile extracts identifiers from a .tf file (placeholder implementation)
func ParseFile(path string) ([]engine.Finding, error) {
    src, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    file, diag := hclsyntax.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
    if diag.HasErrors() {
        return nil, fmt.Errorf("%s: %s", path, diag.Error())
    }
    // TODO: walk AST, apply rules loaded from config
    _ = file
    return nil, nil
}

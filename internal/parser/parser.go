package parser

import (
    "fmt"
    "io/fs"
    "os"
    "path/filepath"

    hcl "github.com/hashicorp/hcl/v2"
    hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"

    "github.com/josdagaro/tfsuit/internal/config"
    "github.com/josdagaro/tfsuit/internal/model"
)

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
func ParseFile(path string, cfg *config.Config) ([]model.Finding, error) {
    src, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    file, diags := hclsyntax.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
    if diags.HasErrors() {
        return nil, fmt.Errorf("%s: %s", path, diags.Error())
    }

    // TODO: Walk AST (file.Body) and apply cfg rules to produce findings
    _ = file

    return nil, nil
}

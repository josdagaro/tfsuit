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

// ParseFile extracts identifiers and evaluates naming rules
func ParseFile(path string, cfg *config.Config) ([]model.Finding, error) {
    src, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    file, diags := hclsyntax.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
    if diags.HasErrors() {
        return nil, fmt.Errorf("%s: %s", path, diags.Error())
    }

    // Convert generic body to concrete syntax body to access Blocks
    syntaxBody, ok := file.Body.(*hclsyntax.Body)
    if !ok {
        return nil, fmt.Errorf("unexpected body type in %s", path)
    }

    var findings []model.Finding

    for _, block := range syntaxBody.Blocks {
        switch block.Type {
        case "variable":
            if len(block.Labels) == 0 {
                continue
            }
            name := block.Labels[0]
            rule := &cfg.Variables
            if rule.IsIgnored(name) {
                continue
            }
            if !rule.Matches(name) {
                findings = append(findings, model.Finding{
                    File:    path,
                    Line:    block.DefRange().Start.Line,
                    Kind:    "variable",
                    Name:    name,
                    Message: fmt.Sprintf("variable '%s' does not match pattern %s", name, rule.Pattern),
                })
            }
        }
    }

    return findings, nil
}

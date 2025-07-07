package engine

import (
    "encoding/json"
    "fmt"

    "github.com/josdagaro/tfsuit/internal/config"
    "github.com/josdagaro/tfsuit/internal/model"
    "github.com/josdagaro/tfsuit/internal/parser"
)

// Scan walks dir, parses files and returns findings
func Scan(dir string, cfg *config.Config) ([]model.Finding, error) {
    files, err := parser.Discover(dir)
    if err != nil {
        return nil, err
    }

    var violations []model.Finding
    for _, f := range files {
        fnds, err := parser.ParseFile(f, cfg)
        if err != nil {
            return err, nil
        }
        violations = append(violations, fnds...)
    }
    return violations, nil
}

// Format serialises findings according to requested format
func Format(f []model.Finding, mode string) string {
    switch mode {
    case "json":
        b, _ := json.MarshalIndent(f, "", "  ")
        return string(b) + "\n"
    default:
        if len(f) == 0 {
            return "\u2705 No naming violations found\n"
        }
        out := "\n\u274C Violations:\n"
        for _, v := range f {
            out += fmt.Sprintf("%s:%d [%s] %s\n", v.File, v.Line, v.Kind, v.Message)
        }
        return out
    }
}

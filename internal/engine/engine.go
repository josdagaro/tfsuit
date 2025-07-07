package engine

import (
    "encoding/json"
    "fmt"

    "github.com/josdagaro/tfsuit/internal/parser"
)

type Finding struct {
    File    string `json:"file"`
    Line    int    `json:"line"`
    Kind    string `json:"kind"`
    Name    string `json:"name"`
    Message string `json:"message"`
}

// Scan walks dir, parses files and returns findings
func Scan(dir string, cfgPath string) ([]Finding, error) {
    // TODO: wire cfg, cache, concurrency
    files, err := parser.Discover(dir)
    if err != nil {
        return nil, err
    }

    var violations []Finding
    for _, f := range files {
        fnds, err := parser.ParseFile(f)
        if err != nil {
            return nil, err
        }
        violations = append(violations, fnds...)
    }
    return violations, nil
}

// Format serialises findings according to requested format
func Format(f []Finding, mode string) string {
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

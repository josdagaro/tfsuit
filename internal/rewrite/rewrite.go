package rewrite

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "regexp"
    "strings"

    "github.com/sergi/go-diff/diffmatchpatch"

    "github.com/josdagaro/tfsuit/internal/config"
    "github.com/josdagaro/tfsuit/internal/parser"
)

type Options struct {
    Write  bool
    DryRun bool
}

// toSnake converts arbitrary name to snake_case deterministic
var nonAlnum = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func toSnake(s string) string {
    s = strings.Trim(nonAlnum.ReplaceAllString(strings.ToLower(s), "_"), "_")
    s = regexp.MustCompile(`_+`).ReplaceAllString(s, "_")
    return s
}

func Run(root string, cfg *config.Config, opt Options) error {
    files, err := parser.Discover(root)
    if err != nil {
        return err
    }
    dmp := diffmatchpatch.New()

    for _, path := range files {
        original, _ := ioutil.ReadFile(path)
        newContent := original

        // simple string replace per finding (same file only)
        findings, _ := parser.ParseFile(path, cfg)
        if len(findings) == 0 {
            continue
        }
        for _, f := range findings {
            fixed := toSnake(f.Name)
            newContent = bytes.ReplaceAll(newContent, []byte(f.Name), []byte(fixed))
        }

        if opt.DryRun {
            if !bytes.Equal(original, newContent) {
                diffs := dmp.DiffMain(string(original), string(newContent), false)
                fmt.Printf("\n--- %s\n", path)
                fmt.Println(dmp.DiffPrettyText(diffs))
            }
        } else if opt.Write {
            if !bytes.Equal(original, newContent) {
                if err := ioutil.WriteFile(path, newContent, 0o644); err != nil {
                    return err
                }
                fmt.Printf("fixed %s\n", path)
            }
        }
    }
    return nil
}

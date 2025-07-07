package engine

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "runtime"
    "sync"
    "strings"
    "sort"

    "github.com/josdagaro/tfsuit/internal/cache"
    "github.com/josdagaro/tfsuit/internal/config"
    "github.com/josdagaro/tfsuit/internal/model"
    "github.com/josdagaro/tfsuit/internal/parser"
)

// Scan walks dir, parses files concurrently, leverages cache and returns findings.
func Scan(dir string, cfg *config.Config) ([]model.Finding, error) {
    files, err := parser.Discover(dir)
    if err != nil {
        return nil, err
    }

    // Load previous cache
    c, _ := cache.Load(dir)

    // Channels and workers
    jobs := make(chan string)
    findingsCh := make(chan []model.Finding)
    wg := sync.WaitGroup{}

    workers := runtime.NumCPU() * 2
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for path := range jobs {
                // Read file and hash
                content, err := ioutil.ReadFile(path)
                if err != nil {
                    continue // skip error; could log
                }
                hash := cache.Hash(content)
                                // Parse file
                res, err := parser.ParseFile(path, cfg)
                if err == nil {
                    findingsCh <- res
                }
                // Update cache entry
                c.PathHashes[path] = hash
            }
        }()
    }

    // Feed jobs
    go func() {
        for _, f := range files {
            jobs <- f
        }
        close(jobs)
    }()

    // Close findings when workers done
    go func() {
        wg.Wait()
        close(findingsCh)
    }()

    var all []model.Finding
    for batch := range findingsCh {
        all = append(all, batch...)
    }

    // Save cache (ignore error silently)
    _ = c.Save(dir)

    return all, nil
}

// Format serialises findings according to the requested format.
// Modes: "pretty" (default) or "json".
// Pretty output sorts by file+line and prints one violation per line.
func Format(f []model.Finding, mode string) string {
    switch mode {
    case "json":
        b, _ := json.MarshalIndent(f, "", "  ")
        return string(b) + "\n"

    default: // pretty
        if len(f) == 0 {
            return "✅ No naming violations found\n"
        }

        // orden estable por archivo y línea
        sort.Slice(f, func(i, j int) bool {
            if f[i].File == f[j].File {
                return f[i].Line < f[j].Line
            }
            return f[i].File < f[j].File
        })

        var sb strings.Builder
        sb.WriteString("\n❌ Violations:\n")

        for _, v := range f {
            fmt.Fprintf(&sb, "%s:%d [%s] %s\n",
                v.File, v.Line, v.Kind, v.Message)
        }
        return sb.String()
    }
}

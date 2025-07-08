package engine

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"runtime"
	"sort"
	"strings"
	"sync"

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

	c, _ := cache.Load(dir) // previous cache

	jobs := make(chan string)
	out := make(chan []model.Finding)
	wg := sync.WaitGroup{}

	workers := runtime.NumCPU() * 2
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				content, err := ioutil.ReadFile(path)
				if err != nil {
					continue
				}
				hash := cache.Hash(content)

				res, err := parser.ParseFile(path, cfg)
				if err == nil && len(res) > 0 {
					out <- res
				}
				c.PathHashes[path] = hash
			}
		}()
	}

	go func() {
		for _, f := range files {
			jobs <- f
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(out)
	}()

	var all []model.Finding
	for batch := range out {
		all = append(all, batch...)
	}

	_ = c.Save(dir)
	return all, nil
}

// Format serialises findings according to the requested format.
// Modes: "pretty" (default), "json", "sarif".
func Format(f []model.Finding, mode string) string {
	switch mode {

	case "json":
		b, _ := json.MarshalIndent(f, "", "  ")
		return string(b) + "\n"

	case "sarif":
		return buildSARIF(f) + "\n"

	default: // pretty
		if len(f) == 0 {
			return "✅ No naming violations found\n"
		}

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

// buildSARIF constructs a minimal SARIF v2.1.0 document.
func buildSARIF(findings []model.Finding) string {
	type (
		artifactLocation struct{ Uri string `json:"uri"` }
		region           struct{ StartLine int `json:"startLine"` }
		physicalLocation struct {
			ArtifactLocation artifactLocation `json:"artifactLocation"`
			Region           region           `json:"region"`
		}
		location struct{ PhysicalLocation physicalLocation `json:"physicalLocation"` }
		message  struct{ Text string `json:"text"` }
		result   struct {
			Level     string     `json:"level"`
			Message   message    `json:"message"`
			Locations []location `json:"locations"`
		}
		driver struct {
			Name           string `json:"name"`
			Version        string `json:"version"`
			InformationURI string `json:"informationUri"`
		}
	)

	sarif := map[string]interface{}{
		"version": "2.1.0",
		"$schema": "https://json.schemastore.org/sarif-2.1.0.json",
		"runs": []interface{}{map[string]interface{}{
			"tool": map[string]interface{}{
				"driver": driver{
					Name:           "tfsuit",
					Version:        "0.1.0",
					InformationURI: "https://github.com/josdagaro/tfsuit",
				},
			},
		}},
	}

	var results []result
	for _, v := range findings {
		results = append(results, result{
			Level:   "error",
			Message: message{Text: v.Message},
			Locations: []location{{
				PhysicalLocation: physicalLocation{
					ArtifactLocation: artifactLocation{Uri: v.File},
					Region:           region{StartLine: v.Line},
				},
			}},
		})
	}
	sarif["runs"].([]interface{})[0].(map[string]interface{})["results"] = results

	b, _ := json.MarshalIndent(sarif, "", "  ")
	return string(b)
}

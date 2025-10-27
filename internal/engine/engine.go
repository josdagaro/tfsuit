package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/josdagaro/tfsuit/internal/cache"
	"github.com/josdagaro/tfsuit/internal/config"
	"github.com/josdagaro/tfsuit/internal/model"
	"github.com/josdagaro/tfsuit/internal/parser"
)

// Scan recorre el dir, parsea concurrentemente, usa caché y devuelve hallazgos.
func Scan(dir string, cfg *config.Config) ([]model.Finding, error) {
	files, err := parser.Discover(dir)
	if err != nil {
		return nil, err
	}

	// Carga caché previo
	c, _ := cache.Load(dir)
	if c.PathHashes == nil {
		c.PathHashes = map[string]string{}
	}

	// 🔒 proteje escrituras al mapa del caché (evita concurrent map writes)
	var cacheMu sync.Mutex

	// Workers y canales
	jobs := make(chan string)
	findingsCh := make(chan []model.Finding)
	wg := sync.WaitGroup{}

	workers := runtime.NumCPU() * 2
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				// Lee archivo y calcula hash
				content, err := os.ReadFile(path)
				if err != nil {
					continue // opcional: log
				}
				hash := cache.Hash(content)

				// Parsea archivo
				res, err := parser.ParseFile(path, cfg)
				if err == nil {
					findingsCh <- res
				}

				// ✅ actualización del caché (protegida)
				cacheMu.Lock()
				c.PathHashes[path] = hash
				cacheMu.Unlock()
			}
		}()
	}

	// Alimenta trabajos
	go func() {
		for _, f := range files {
			jobs <- f
		}
		close(jobs)
	}()

	// Cierra el canal de resultados al terminar los workers
	go func() {
		wg.Wait()
		close(findingsCh)
	}()

	var all []model.Finding
	for batch := range findingsCh {
		all = append(all, batch...)
	}

	// Guarda caché (una sola vez, ya en secuencia)
	_ = c.Save(dir)

	return all, nil
}

// Format serializa hallazgos según el formato.
// Modos: "pretty" (default), "json", "sarif".
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

// buildSARIF construye un SARIF v2.1.0 mínimo.
func buildSARIF(findings []model.Finding) string {
	type (
		artifactLocation struct {
			Uri string `json:"uri"`
		}
		region struct {
			StartLine int `json:"startLine"`
		}
		physicalLocation struct {
			ArtifactLocation artifactLocation `json:"artifactLocation"`
			Region           region           `json:"region"`
		}
		location struct {
			PhysicalLocation physicalLocation `json:"physicalLocation"`
		}
		message struct {
			Text string `json:"text"`
		}
		result struct {
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
					Version:        "1.x", // opcional: puedes inyectar versión aquí si lo deseas
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

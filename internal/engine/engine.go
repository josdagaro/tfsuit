// File: internal/engine/engine.go
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

// Scan recorre el directorio, parsea archivos .tf en paralelo, usa cache
// y devuelve todos los hallazgos como []model.Finding.
func Scan(dir string, cfg *config.Config) ([]model.Finding, error) {
	// Descubre archivos Terraform
	files, err := parser.Discover(dir)
	if err != nil {
		return nil, err
	}

	// Carga/asegura el cache
	c, _ := cache.Load(dir)
	if c.PathHashes == nil {
		c.PathHashes = map[string]string{}
	}

	var cacheMu sync.Mutex

	// Canal de trabajos y resultados
	workers := runtime.GOMAXPROCS(0) // nº de CPU lógicas
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan string, workers*2)
	findingsCh := make(chan []model.Finding, workers*2)

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for path := range jobs {
				// Lee contenido para hashear y cachear
				content, err := os.ReadFile(path)
				if err != nil {
					// si un archivo falla, seguimos con el resto
					continue
				}
				hash := cache.Hash(content)

				// Parsea el archivo → hallazgos
				res, err := parser.ParseFile(path, cfg)
				if err == nil && len(res) > 0 {
					findingsCh <- res
				}

				// Actualiza cache protegido
				cacheMu.Lock()
				c.PathHashes[path] = hash
				cacheMu.Unlock()
			}
		}()
	}

	// Feeder de trabajos
	go func() {
		for _, f := range files {
			jobs <- f
		}
		close(jobs)
	}()

	// Cerramos resultados cuando terminen los workers
	go func() {
		wg.Wait()
		close(findingsCh)
	}()

	// Agregamos todo en un slice
	var all []model.Finding
	for batch := range findingsCh {
		all = append(all, batch...)
	}

	// Guardamos cache (opcionalmente ignora error)
	_ = c.Save(dir)

	return all, nil
}

// Format serializa hallazgos en el formato solicitado.
// Soporta: "pretty" (por defecto), "json" y "sarif".
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

// buildSARIF construye un SARIF 2.1.0 mínimo con los hallazgos.
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
					Version:        "1.x", // opcional: inyecta versión real si la tienes
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

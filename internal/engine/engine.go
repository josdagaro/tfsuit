package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/josdagaro/tfsuit/internal/cache"
	"github.com/josdagaro/tfsuit/internal/config"
	"github.com/josdagaro/tfsuit/internal/model"
	"github.com/josdagaro/tfsuit/internal/parser"
)

// ScanStats captura info del escaneo para mejorar mensajes en modo "pretty".
type ScanStats struct {
	Files    int
	Duration time.Duration
}

// Scan recorre el dir, parsea concurrentemente, usa caché y devuelve hallazgos + estadísticas.
func Scan(dir string, cfg *config.Config) ([]model.Finding, ScanStats, error) {
	start := time.Now()

	files, err := parser.Discover(dir)
	if err != nil {
		return nil, ScanStats{}, err
	}
	stats := ScanStats{Files: len(files)}

	// Carga caché previo
	c, _ := cache.Load(dir)
	if c.PathHashes == nil {
		c.PathHashes = map[string]string{}
	}

	// 🔒 protege escrituras al mapa del caché (evita concurrent map writes)
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

	stats.Duration = time.Since(start)
	return all, stats, nil
}

// Format serializa hallazgos según el formato.
// Modos: "pretty" (default), "json", "sarif".
func Format(f []model.Finding, mode string, stats *ScanStats) string {
	switch mode {

	case "json":
		b, _ := json.MarshalIndent(f, "", "  ")
		return string(b) + "\n"

	case "sarif":
		return buildSARIF(f) + "\n"

	default: // pretty
		// Resumen cuando NO hay violaciones
		if len(f) == 0 {
			if stats != nil {
				d := stats.Duration.Truncate(10 * time.Millisecond)
				word := "files"
				if stats.Files == 1 {
					word = "file"
				}
				return fmt.Sprintf("✅ No naming violations found — scanned %d %s in %s\n",
					stats.Files, word, d)
			}
			return "✅ No naming violations found\n"
		}

		// Ordenamos para una salida estable
		sort.Slice(f, func(i, j int) bool {
			if f[i].File == f[j].File {
				return f[i].Line < f[j].Line
			}
			return f[i].File < f[j].File
		})

		// Contadores para el resumen
		byKind := map[string]int{}
		fileSet := map[string]struct{}{}
		for _, v := range f {
			byKind[v.Kind]++
			fileSet[v.File] = struct{}{}
		}

		// Impresión de violaciones
		var sb strings.Builder
		sb.WriteString("\n❌ Violations:\n")
		for _, v := range f {
			fmt.Fprintf(&sb, "%s:%d [%s] %s\n",
				v.File, v.Line, v.Kind, v.Message)
		}

		// Resumen al final (cuando SÍ hay violaciones)
		if stats != nil {
			d := stats.Duration.Truncate(10 * time.Millisecond)

			// Desglose por tipo en orden legible
			order := []string{"variable", "output", "module", "data", "resource"}
			var parts []string
			for _, k := range order {
				if n := byKind[k]; n > 0 {
					name := k
					if n != 1 {
						name += "s"
					}
					parts = append(parts, fmt.Sprintf("%d %s", n, name))
				}
			}

			sb.WriteString("\n— ")
			sb.WriteString(fmt.Sprintf("%d violations", len(f)))
			if len(parts) > 0 {
				sb.WriteString(fmt.Sprintf(" (%s)", strings.Join(parts, ", ")))
			}
			sb.WriteString(fmt.Sprintf(" across %d/%d files in %s\n",
				len(fileSet), stats.Files, d))
		} else {
			// Sin stats disponibles
			sb.WriteString(fmt.Sprintf("\n— %d violations\n", len(f)))
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
					Version:        "1.x", // opcional: puedes inyectar versión real si lo deseas
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

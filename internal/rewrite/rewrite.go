package rewrite

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/josdagaro/tfsuit/internal/config"
	"github.com/josdagaro/tfsuit/internal/model"
	"github.com/josdagaro/tfsuit/internal/parser"
)

/* -------------------------------------------------------------------------- */
/* Types & helpers                                                            */
/* -------------------------------------------------------------------------- */

type Options struct{ Write, DryRun bool }

var nonAlnum = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func toSnake(s string) string {
	s = strings.Trim(nonAlnum.ReplaceAllString(strings.ToLower(s), "_"), "_")
	return regexp.MustCompile(`_+`).ReplaceAllString(s, "_")
}

type rename struct{ Old, New string }

/* -------------------------------------------------------------------------- */
/* Entry point                                                                */
/* -------------------------------------------------------------------------- */

func Run(root string, cfg *config.Config, opt Options) error {
	files, err := collectTfFiles(root)
	if err != nil {
		return err
	}

	fileRen := map[string][]rename{}
	globalRen := map[string]string{} // old → new

	/* ---------- 1️⃣  primera pasada: detectar violaciones ---------------- */

	for _, path := range files {
		src, _ := ioutil.ReadFile(path)
		file, diags := hclsyntax.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			continue
		}

		body := file.Body.(*hclsyntax.Body)
		for _, b := range body.Blocks {

			switch b.Type {

			case "variable", "output", "module":
				if len(b.Labels) == 0 {
					continue
				}
				old := b.Labels[0]
				rule := map[string]*config.Rule{
					"variable": &cfg.Variables,
					"output":   &cfg.Outputs,
					"module":   &cfg.Modules,
				}[b.Type]
				if rule.IsIgnored(old) || rule.Matches(old) {
					continue
				}
				newName := toSnake(old)
				fileRen[path] = append(fileRen[path], rename{old, newName})
				globalRen[old] = newName

			case "resource":
				if len(b.Labels) < 2 {
					continue
				}
				old := b.Labels[1]
				if cfg.Resources.IsIgnored(old) || cfg.Resources.Matches(old) {
					continue
				}
				newName := toSnake(old)
				fileRen[path] = append(fileRen[path], rename{old, newName})
				globalRen[old] = newName
			}
		}
	}

	if len(globalRen) == 0 {
		fmt.Println("✅ No fixes needed")
		return nil
	}

	/* ---------- 2️⃣  regex para refs cruzadas (.old  y  module.old) ------- */

	alts := make([]string, 0, len(globalRen))
	for o := range globalRen {
		alts = append(alts, regexp.QuoteMeta(o))
	}
	sort.Slice(alts, func(i, j int) bool { return len(alts[i]) > len(alts[j]) })

	// Grupo 1 = via `module.<old>`   Grupo 2 = via `.old`
	crossRe := regexp.MustCompile(`(?:\bmodule\.(?P<mod>` + strings.Join(alts, "|") +
		`)\b|\.(?P<dot>` + strings.Join(alts, "|") + `)\b)`)

	dmp := diffmatchpatch.New()

	/* ---------- 3️⃣  reescritura por archivo ----------------------------- */

	for _, path := range files {
		orig, _ := ioutil.ReadFile(path)
		mod := orig

		// 3a. renombres locales
		for _, rn := range fileRen[path] {
			mod = bytes.ReplaceAll(mod, []byte(rn.Old), []byte(rn.New))
		}

		// 3b. referencias cruzadas
		mod = crossRe.ReplaceAllFunc(mod, func(b []byte) []byte {
			m := crossRe.FindSubmatch(b)
			var old string
			if len(m) >= 2 && len(m[1]) > 0 { // module.<old>
				old = string(m[1])
			} else if len(m) >= 3 && len(m[2]) > 0 { // .old
				old = string(m[2])
			}
			if nn, ok := globalRen[old]; ok {
				return bytes.Replace(b, []byte(old), []byte(nn), 1)
			}
			return b
		})

		if bytes.Equal(orig, mod) {
			continue
		}

		if opt.DryRun {
			fmt.Printf("\n--- %s\n", path)
			diff := dmp.DiffMain(string(orig), string(mod), false)
			fmt.Print(dmp.DiffPrettyText(diff))
		} else if opt.Write {
			if err := ioutil.WriteFile(path, mod, 0o644); err != nil {
				return err
			}
			fmt.Printf("fixed %s\n", path)
		}
	}

	return nil
}

/* -------------------------------------------------------------------------- */
/* Helpers                                                                    */
/* -------------------------------------------------------------------------- */

func collectTfFiles(root string) ([]string, error) {
	var out []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == ".terraform" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(info.Name(), ".tf") {
			out = append(out, p)
		}
		return nil
	})
	return out, err
}

func ScanFileAfterFix(path string, cfg *config.Config) ([]model.Finding, error) {
	return parser.ParseFile(path, cfg)
}

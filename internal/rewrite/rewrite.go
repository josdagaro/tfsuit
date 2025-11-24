package rewrite

import (
	"bytes"
	"errors"
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
	cty "github.com/zclconf/go-cty/cty"

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

type providerInsertion struct {
	Offset  int
	Payload string
}

var errNoProviderDefinitions = errors.New("no provider configurations defined")

/* -------------------------------------------------------------------------- */
/* Entry point                                                                */
/* -------------------------------------------------------------------------- */

func Run(root string, cfg *config.Config, opt Options) error {
	files, err := collectTfFiles(root)
	if err != nil {
		return err
	}

	provCatalog, err := analyzeProviders(files)
	if err != nil {
		return err
	}

	fileRen := map[string][]rename{}
	globalRen := map[string]string{} // old → new
	providerFixes := map[string][]providerInsertion{}
	// métricas para el resumen final
	var (
		declRenames         int // cantidad de etiquetas a renombrar (declaraciones)
		filesWithDecl       int // archivos que contienen al menos un renombre de declaración
		filesChanged        int // archivos que cambian (dry-run o write)
		xrefHits            int // cantidad de referencias cruzadas reemplazadas
		providerAssignments int // cantidad de providers inyectados
	)
	hasProviderFixes := false

	requireProvider := map[string]bool{
		"module":   cfg.Modules.RequiresProvider(),
		"resource": cfg.Resources.RequiresProvider(),
	}
	if cfg.Data != nil {
		requireProvider["data"] = cfg.Data.RequiresProvider()
	}

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

			case "variable", "output":
				if len(b.Labels) == 0 {
					continue
				}
				old := b.Labels[0]
				rule := map[string]*config.Rule{
					"variable": &cfg.Variables,
					"output":   &cfg.Outputs,
				}[b.Type]
				if rule.IsIgnored(old) || rule.Matches(old) {
					continue
				}
				newName := toSnake(old)
				fileRen[path] = append(fileRen[path], rename{old, newName})
				globalRen[old] = newName

			case "module":
				if len(b.Labels) == 0 {
					continue
				}
				old := b.Labels[0]
				if !(cfg.Modules.IsIgnored(old) || cfg.Modules.Matches(old)) {
					newName := toSnake(old)
					fileRen[path] = append(fileRen[path], rename{old, newName})
					globalRen[old] = newName
				}

				if requireProvider["module"] && needsProviderAssignment(b, "module") {
					if err := scheduleProviderFix(path, src, b, "module", "", provCatalog, providerFixes, root); err != nil {
						return err
					}
					hasProviderFixes = true
				}

			case "resource":
				if len(b.Labels) < 2 {
					continue
				}
				old := b.Labels[1]
				if !(cfg.Resources.IsIgnored(old) || cfg.Resources.Matches(old)) {
					newName := toSnake(old)
					fileRen[path] = append(fileRen[path], rename{old, newName})
					globalRen[old] = newName
				}

				if requireProvider["resource"] && needsProviderAssignment(b, "resource") {
					pref := providerTypeFromBlock(b)
					if err := scheduleProviderFix(path, src, b, "resource", pref, provCatalog, providerFixes, root); err != nil {
						return err
					}
					hasProviderFixes = true
				}

			case "data":
				if len(b.Labels) < 2 {
					continue
				}
				old := b.Labels[1]
				if cfg.Data == nil {
					continue
				}
				if !(cfg.Data.IsIgnored(old) || cfg.Data.Matches(old)) {
					newName := toSnake(old)
					fileRen[path] = append(fileRen[path], rename{old, newName})
					globalRen[old] = newName
				}

				if requireProvider["data"] && needsProviderAssignment(b, "data") {
					pref := providerTypeFromBlock(b)
					if err := scheduleProviderFix(path, src, b, "data", pref, provCatalog, providerFixes, root); err != nil {
						return err
					}
					hasProviderFixes = true
				}
			}
		}
	}

	if len(globalRen) == 0 && !hasProviderFixes {
		// Alinea el comportamiento con la solicitud de un resumen al final
		if opt.DryRun {
			fmt.Println("✅ No fixes needed")
			fmt.Println("Summary (dry-run): 0 labels to rename; 0 files would change; 0 cross-references.")
		} else if opt.Write {
			fmt.Println("✅ Nothing to change")
			fmt.Println("Summary: 0 labels renamed; 0 files updated; 0 cross-references.")
		}
		return nil
	}

	/* ---------- 2️⃣  regex para refs cruzadas (.old  y  module.old) ------- */

	var crossRe *regexp.Regexp
	if len(globalRen) > 0 {
		alts := make([]string, 0, len(globalRen))
		for o := range globalRen {
			alts = append(alts, regexp.QuoteMeta(o))
		}
		sort.Slice(alts, func(i, j int) bool { return len(alts[i]) > len(alts[j]) })

		// Grupo 1 = via `module.<old>`   Grupo 2 = via `.old`
		crossRe = regexp.MustCompile(`(?:\bmodule\.(?P<mod>` + strings.Join(alts, "|") +
			`)\b|\.(?P<dot>` + strings.Join(alts, "|") + `)\b)`)
	}

	dmp := diffmatchpatch.New()

	/* ---------- 3️⃣  reescritura por archivo ----------------------------- */

	for _, path := range files {
		orig, _ := ioutil.ReadFile(path)
		mod := orig

		// 3a. providers faltantes
		if ins := providerFixes[path]; len(ins) > 0 {
			sort.Slice(ins, func(i, j int) bool { return ins[i].Offset > ins[j].Offset })
			for _, fix := range ins {
				mod = insertText(mod, fix.Offset, fix.Payload)
			}
			providerAssignments += len(ins)
		}

		// 3b. renombres locales
		if len(fileRen[path]) > 0 {
			filesWithDecl++
			declRenames += len(fileRen[path])
		}
		for _, rn := range fileRen[path] {
			mod = bytes.ReplaceAll(mod, []byte(rn.Old), []byte(rn.New))
		}

		// 3c. referencias cruzadas
		if crossRe != nil {
			mod = crossRe.ReplaceAllFunc(mod, func(b []byte) []byte {
				m := crossRe.FindSubmatch(b)
				var old string
				if len(m) >= 2 && len(m[1]) > 0 { // module.<old>
					old = string(m[1])
				} else if len(m) >= 3 && len(m[2]) > 0 { // .old
					old = string(m[2])
				}
				if nn, ok := globalRen[old]; ok {
					xrefHits++
					return bytes.Replace(b, []byte(old), []byte(nn), 1)
				}
				return b
			})
		}

		if bytes.Equal(orig, mod) {
			continue
		}

		if opt.DryRun {
			fmt.Printf("\n--- %s\n", path)
			diff := dmp.DiffMain(string(orig), string(mod), false)
			fmt.Print(dmp.DiffPrettyText(diff))
			filesChanged++
		} else if opt.Write {
			if err := ioutil.WriteFile(path, mod, 0o644); err != nil {
				return err
			}
			fmt.Printf("fixed %s\n", path)
			filesChanged++
		}
	}

	// resumen final
	if opt.DryRun {
		fmt.Printf("\nSummary (dry-run): %d labels to rename across %d files; would update %d files; %d cross-references.",
			declRenames, filesWithDecl, filesChanged, xrefHits)
		if providerAssignments > 0 {
			fmt.Printf(" Would add %d provider assignments.", providerAssignments)
		}
		fmt.Printf("\n")
	} else if opt.Write {
		fmt.Printf("\nSummary: renamed %d labels across %d files; updated %d files; %d cross-references.",
			declRenames, filesWithDecl, filesChanged, xrefHits)
		if providerAssignments > 0 {
			fmt.Printf(" Added %d provider assignments.", providerAssignments)
		}
		fmt.Printf("\n")
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

func needsProviderAssignment(block *hclsyntax.Block, kind string) bool {
	switch kind {
	case "module":
		attr, ok := block.Body.Attributes["providers"]
		if !ok {
			return true
		}
		if obj, ok := attr.Expr.(*hclsyntax.ObjectConsExpr); ok {
			return len(obj.Items) == 0
		}
		return false
	case "resource", "data":
		_, ok := block.Body.Attributes["provider"]
		return !ok
	default:
		return false
	}
}

func providerTypeFromBlock(block *hclsyntax.Block) string {
	if len(block.Labels) == 0 {
		return ""
	}
	raw := block.Labels[0]
	if idx := strings.IndexRune(raw, '_'); idx >= 0 {
		return raw[:idx]
	}
	return raw
}

func scheduleProviderFix(path string, src []byte, block *hclsyntax.Block, kind, preferredType string, catalog *providerCatalog, providerFixes map[string][]providerInsertion, root string) error {
	cfg, err := catalog.choose(preferredType)
	if err != nil {
		if errors.Is(err, errNoProviderDefinitions) {
			if err := ensureProvidersFile(root); err != nil {
				return err
			}
			return fmt.Errorf("tfsuit fix requires at least one provider configuration with an alias (see %s); define it and rerun", filepath.Join(root, "providers.tf"))
		}
		return err
	}
	offset := block.CloseBraceRange.Start.Byte
	indent := indentAt(src, offset)
	payload := renderProviderPayload(kind, indent, cfg)
	providerFixes[path] = append(providerFixes[path], providerInsertion{
		Offset:  offset,
		Payload: payload,
	})
	return nil
}

func indentAt(src []byte, offset int) string {
	if offset > len(src) {
		offset = len(src)
	}
	i := offset - 1
	for i >= 0 {
		if src[i] == '\n' || src[i] == '\r' {
			break
		}
		i--
	}
	start := i + 1
	j := start
	for j < offset {
		if src[j] != ' ' && src[j] != '\t' {
			break
		}
		j++
	}
	return string(src[start:j])
}

func renderProviderPayload(kind, indent string, cfg *providerConfig) string {
	ref := cfg.Address
	switch kind {
	case "module":
		return fmt.Sprintf("\n%s  providers = {\n%s    %s = %s\n%s  }\n%s",
			indent, indent, cfg.Type, ref, indent, indent)
	case "resource", "data":
		return fmt.Sprintf("\n%s  provider = %s\n%s", indent, ref, indent)
	default:
		return ""
	}
}

func insertText(src []byte, offset int, payload string) []byte {
	if offset < 0 {
		offset = 0
	}
	if offset > len(src) {
		offset = len(src)
	}
	buf := make([]byte, 0, len(src)+len(payload))
	buf = append(buf, src[:offset]...)
	buf = append(buf, payload...)
	buf = append(buf, src[offset:]...)
	return buf
}

func ensureProvidersFile(root string) error {
	path := filepath.Join(root, "providers.tf")
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	content := `# This file was created by tfsuit fix.
# Define at least one provider with an alias so automatic fixes can assign it.
# Example:
# provider "aws" {
#   alias  = "primary"
#   region = "us-east-1"
# }
`
	return os.WriteFile(path, []byte(content), 0o644)
}

type providerCatalog struct {
	entries  []*providerConfig
	byAddr   map[string]*providerConfig
	defCount int
}

type providerConfig struct {
	Address string
	Type    string
	Alias   string
	Count   int
	Order   int
	Defined bool
}

func analyzeProviders(files []string) (*providerCatalog, error) {
	catalog := &providerCatalog{
		byAddr: map[string]*providerConfig{},
	}
	order := 0

	for _, path := range files {
		src, err := ioutil.ReadFile(path)
		if err != nil {
			continue
		}
		file, diags := hclsyntax.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			continue
		}
		body, ok := file.Body.(*hclsyntax.Body)
		if !ok {
			continue
		}

		for _, b := range body.Blocks {
			switch b.Type {
			case "provider":
				if len(b.Labels) == 0 {
					continue
				}
				typ := b.Labels[0]
				alias := ""
				if attr, ok := b.Body.Attributes["alias"]; ok {
					if val, diags := attr.Expr.Value(nil); diags == nil || !diags.HasErrors() {
						if val.Type() == cty.String {
							alias = val.AsString()
						}
					}
				}
				catalog.recordDefinition(typ, alias, order)
				order++
			case "module":
				if attr, ok := b.Body.Attributes["providers"]; ok {
					if obj, ok := attr.Expr.(*hclsyntax.ObjectConsExpr); ok {
						for _, item := range obj.Items {
							if ref := providerRefFromExpr(item.ValueExpr); ref != "" {
								catalog.recordUsage(ref)
							}
						}
					}
				}
			case "resource", "data":
				if attr, ok := b.Body.Attributes["provider"]; ok {
					if ref := providerRefFromExpr(attr.Expr); ref != "" {
						catalog.recordUsage(ref)
					}
				}
			}
		}
	}

	return catalog, nil
}

func (c *providerCatalog) recordDefinition(typ, alias string, order int) {
	addr := typ
	if alias != "" {
		addr = typ + "." + alias
	}
	entry := c.ensure(addr, typ, alias)
	if !entry.Defined {
		entry.Defined = true
		entry.Order = order
		c.defCount++
	}
}

func (c *providerCatalog) recordUsage(ref string) {
	addr, typ, alias := splitProviderRef(ref)
	entry := c.ensure(addr, typ, alias)
	entry.Count++
}

func (c *providerCatalog) ensure(addr, typ, alias string) *providerConfig {
	if cfg, ok := c.byAddr[addr]; ok {
		return cfg
	}
	cfg := &providerConfig{
		Address: addr,
		Type:    typ,
		Alias:   alias,
		Order:   len(c.entries),
	}
	c.byAddr[addr] = cfg
	c.entries = append(c.entries, cfg)
	return cfg
}

func (c *providerCatalog) choose(preferredType string) (*providerConfig, error) {
	if c.defCount == 0 {
		return nil, errNoProviderDefinitions
	}

	candidates := c.filterByType(preferredType)
	if len(candidates) == 0 && preferredType != "" {
		candidates = c.filterByType("")
	}
	if len(candidates) == 0 {
		return nil, errNoProviderDefinitions
	}

	withAlias := filterDefinedWithAlias(candidates)
	if len(withAlias) > 0 {
		candidates = withAlias
	} else {
		withDefinition := filterDefined(candidates)
		if len(withDefinition) > 0 {
			candidates = withDefinition
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Count == candidates[j].Count {
			return candidates[i].Order < candidates[j].Order
		}
		return candidates[i].Count > candidates[j].Count
	})
	return candidates[0], nil
}

func (c *providerCatalog) filterByType(typ string) []*providerConfig {
	if typ == "" {
		return append([]*providerConfig(nil), c.entries...)
	}
	var out []*providerConfig
	for _, cfg := range c.entries {
		if cfg.Type == typ {
			out = append(out, cfg)
		}
	}
	return out
}

func filterDefinedWithAlias(list []*providerConfig) []*providerConfig {
	var out []*providerConfig
	for _, cfg := range list {
		if cfg.Defined && cfg.Alias != "" {
			out = append(out, cfg)
		}
	}
	return out
}

func filterDefined(list []*providerConfig) []*providerConfig {
	var out []*providerConfig
	for _, cfg := range list {
		if cfg.Defined {
			out = append(out, cfg)
		}
	}
	return out
}

func providerRefFromExpr(expr hclsyntax.Expression) string {
	switch e := expr.(type) {
	case *hclsyntax.ScopeTraversalExpr:
		return traversalToProviderRef(e.Traversal)
	case *hclsyntax.RelativeTraversalExpr:
		return traversalToProviderRef(e.Traversal)
	case *hclsyntax.TemplateExpr:
		if len(e.Parts) == 1 {
			if lit, ok := e.Parts[0].(*hclsyntax.TemplateWrapExpr); ok {
				return providerRefFromExpr(lit.Wrapped)
			}
		}
	}
	return ""
}

func traversalToProviderRef(tr hcl.Traversal) string {
	var parts []string
	for _, step := range tr {
		switch v := step.(type) {
		case hcl.TraverseRoot:
			parts = append(parts, v.Name)
		case hcl.TraverseAttr:
			parts = append(parts, v.Name)
		default:
			return ""
		}
	}
	if len(parts) == 0 || len(parts) > 2 {
		return ""
	}
	return strings.Join(parts, ".")
}

func splitProviderRef(ref string) (addr, typ, alias string) {
	parts := strings.Split(ref, ".")
	if len(parts) == 0 {
		return ref, ref, ""
	}
	typ = parts[0]
	if len(parts) > 1 {
		alias = parts[1]
	}
	if alias != "" {
		addr = typ + "." + alias
	} else {
		addr = typ
	}
	return
}

func ScanFileAfterFix(path string, cfg *config.Config) ([]model.Finding, error) {
	return parser.ParseFile(path, cfg)
}

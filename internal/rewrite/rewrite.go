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

type Options struct {
	Write    bool
	DryRun   bool
	FixKinds map[string]bool
}

func (opt Options) allows(kind string) bool {
	if len(opt.FixKinds) == 0 {
		return true
	}
	return opt.FixKinds[kind]
}

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

type providerSelection struct {
	Alias string
	Type  string
}

type fileRename struct {
	Old string
	New string
}

var (
	errNoProviderDefinitions = errors.New("no provider configurations defined")
	errNoProviderInScope     = errors.New("no provider available in scope")
)

/* -------------------------------------------------------------------------- */
/* Entry point                                                                */
/* -------------------------------------------------------------------------- */

func Run(root string, cfg *config.Config, opt Options) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	root = absRoot

	files, err := collectTfFiles(root)
	if err != nil {
		return err
	}

	resolver, err := buildProviderResolver(root, files)
	if err != nil {
		return err
	}

	fileRen := map[string][]rename{}
	globalRen := map[string]string{} // old → new
	providerFixes := map[string][]providerInsertion{}
	var pendingFileRenames []fileRename
	if cfg.Files != nil {
	if cfg.Files != nil && opt.allows("file") {
		pendingFileRenames = planFileRenames(files, cfg.Files)
	}
	}
	// métricas para el resumen final
	var (
		declRenames         int // cantidad de etiquetas a renombrar (declaraciones)
		filesWithDecl       int // archivos que contienen al menos un renombre de declaración
		filesChanged        int // archivos que cambian (dry-run o write)
		xrefHits            int // cantidad de referencias cruzadas reemplazadas
		providerAssignments int // cantidad de providers inyectados
		fileRenameCount     int // cantidad de archivos renombrados
	)
	hasProviderFixes := false
	hasFileRenames := len(pendingFileRenames) > 0

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
				if !opt.allows(b.Type) {
					continue
				}
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
				if !opt.allows("module") {
					continue
				}
				if len(b.Labels) == 0 {
					continue
				}
				old := b.Labels[0]
				if !(cfg.Modules.IsIgnored(old) || cfg.Modules.Matches(old)) {
					newName := toSnake(old)
					fileRen[path] = append(fileRen[path], rename{old, newName})
					globalRen[old] = newName
				}

				if requireProvider["module"] && opt.allows("module") && needsProviderAssignment(b, "module") {
					if err := scheduleProviderFix(path, src, b, "module", "", resolver, providerFixes, root); err != nil {
						return err
					}
					hasProviderFixes = true
				}

			case "resource":
				if !opt.allows("resource") {
					continue
				}
				if len(b.Labels) < 2 {
					continue
				}
				old := b.Labels[1]
				if !(cfg.Resources.IsIgnored(old) || cfg.Resources.Matches(old)) {
					newName := toSnake(old)
					fileRen[path] = append(fileRen[path], rename{old, newName})
					globalRen[old] = newName
				}

				if requireProvider["resource"] && opt.allows("resource") && needsProviderAssignment(b, "resource") {
					pref := providerTypeFromBlock(b)
					if err := scheduleProviderFix(path, src, b, "resource", pref, resolver, providerFixes, root); err != nil {
						return err
					}
					hasProviderFixes = true
				}

			case "data":
				if !opt.allows("data") {
					continue
				}
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

				if requireProvider["data"] && opt.allows("data") && needsProviderAssignment(b, "data") {
					pref := providerTypeFromBlock(b)
					if err := scheduleProviderFix(path, src, b, "data", pref, resolver, providerFixes, root); err != nil {
						return err
					}
					hasProviderFixes = true
				}
			}
		}
	}

	if len(globalRen) == 0 && !hasProviderFixes && !hasFileRenames {
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
	if len(pendingFileRenames) > 0 {
		for _, fr := range pendingFileRenames {
			if opt.DryRun {
				fmt.Printf("rename %s -> %s\n", fr.Old, fr.New)
				fileRenameCount++
			} else if opt.Write {
				if err := os.Rename(fr.Old, fr.New); err != nil {
					return err
				}
				fmt.Printf("renamed %s -> %s\n", fr.Old, fr.New)
				fileRenameCount++
				filesChanged++
			}
		}
	}

	if opt.DryRun {
		fmt.Printf("\nSummary (dry-run): %d labels to rename across %d files; would update %d files; %d cross-references.",
			declRenames, filesWithDecl, filesChanged, xrefHits)
		if providerAssignments > 0 {
			fmt.Printf(" Would add %d provider assignments.", providerAssignments)
		}
		if fileRenameCount > 0 {
			fmt.Printf(" Would rename %d files.", fileRenameCount)
		}
		fmt.Printf("\n")
	} else if opt.Write {
		fmt.Printf("\nSummary: renamed %d labels across %d files; updated %d files; %d cross-references.",
			declRenames, filesWithDecl, filesChanged, xrefHits)
		if providerAssignments > 0 {
			fmt.Printf(" Added %d provider assignments.", providerAssignments)
		}
		if fileRenameCount > 0 {
			fmt.Printf(" Renamed %d files.", fileRenameCount)
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

func scheduleProviderFix(path string, src []byte, block *hclsyntax.Block, kind, preferredType string, resolver *providerResolver, providerFixes map[string][]providerInsertion, root string) error {
	dir := filepath.Dir(path)
	offset := block.CloseBraceRange.Start.Byte
	indent := indentAt(src, offset)

	if kind == "module" {
		if aliases := resolver.requiredAliasesForModule(path, block); len(aliases) > 0 {
			payload := renderModuleProvidersPayload(indent, aliases)
			providerFixes[path] = append(providerFixes[path], providerInsertion{
				Offset:  offset,
				Payload: payload,
			})
			return nil
		}
	}

	sel, err := resolver.resolve(dir, preferredType)
	if err != nil {
		if errors.Is(err, errNoProviderDefinitions) {
			if err := ensureProvidersFile(root); err != nil {
				return err
			}
			return fmt.Errorf("tfsuit fix requires at least one provider configuration with an alias (see %s); define it and rerun", filepath.Join(root, "providers.tf"))
		}
		return err
	}
	payload := renderProviderPayload(kind, indent, sel)
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

func renderProviderPayload(kind, indent string, sel providerSelection) string {
	ref := sel.Alias
	switch kind {
	case "module":
		key := sel.Type
		if key == "" {
			key = sel.Alias
		}
		return fmt.Sprintf("\n%s  providers = {\n%s    %s = %s\n%s  }\n%s",
			indent, indent, key, ref, indent, indent)
	case "resource", "data":
		return fmt.Sprintf("\n%s  provider = %s\n%s", indent, ref, indent)
	default:
		return ""
	}
}

func renderModuleProvidersPayload(indent string, aliases []string) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(indent)
	sb.WriteString("  providers = {\n")
	for _, alias := range aliases {
		sb.WriteString(indent)
		sb.WriteString("    ")
		sb.WriteString(alias)
		sb.WriteString(" = ")
		sb.WriteString(alias)
		sb.WriteString("\n")
	}
	sb.WriteString(indent)
	sb.WriteString("  }\n")
	sb.WriteString(indent)
	return sb.String()
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

func planFileRenames(files []string, rule *config.Rule) []fileRename {
	if rule == nil {
		return nil
	}
	existing := make(map[string]struct{}, len(files))
	for _, path := range files {
		existing[path] = struct{}{}
	}

	var renames []fileRename
	for _, path := range files {
		base := filepath.Base(path)
		if rule.IsIgnored(base) || rule.Matches(base) {
			continue
		}
		dir := filepath.Dir(path)
		ext := strings.ToLower(filepath.Ext(base))
		name := strings.TrimSuffix(base, filepath.Ext(base))
		newName := toSnake(name)
		if newName == "" {
			newName = "file"
		}
		if ext == "" {
			ext = ".tf"
		}
		candidate := filepath.Join(dir, newName+ext)
		delete(existing, path)
		suffix := 1
		for {
			if _, taken := existing[candidate]; !taken {
				break
			}
			candidate = filepath.Join(dir, fmt.Sprintf("%s_%d%s", newName, suffix, ext))
			suffix++
		}
		existing[candidate] = struct{}{}
		if candidate == path {
			continue
		}
		renames = append(renames, fileRename{Old: path, New: candidate})
	}
	return renames
}

type providerResolver struct {
	root       string
	scopes     map[string]*providerScope
	aliasUsage map[string]int
	moduleDirs []string
	hasDefs    bool
}

type providerScope struct {
	Path    string
	Aliases map[string]*providerAlias
	Allowed map[string]struct{}
}

type providerAlias struct {
	Name    string
	Type    string
	Defined bool
}

func buildProviderResolver(root string, files []string) (*providerResolver, error) {
	resolver := &providerResolver{
		root:       root,
		scopes:     map[string]*providerScope{},
		aliasUsage: map[string]int{},
	}

	type providerDef struct {
		dir   string
		alias string
	}

	type moduleCall struct {
		parentDir string
		childDir  string
		providers map[string]string
	}

	var defs []providerDef
	var calls []moduleCall
	moduleDirSet := map[string]struct{}{}

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

		dir := filepath.Dir(path)

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
				name := buildProviderAliasName(typ, alias)
				defs = append(defs, providerDef{dir: dir, alias: name})
				resolver.ensureAlias(name)
				resolver.hasDefs = true

			case "resource", "data":
				if attr, ok := b.Body.Attributes["provider"]; ok {
					if ref := providerRefFromExpr(attr.Expr); ref != "" {
						resolver.recordUsage(ref)
					}
				}

			case "module":
				call := moduleCall{parentDir: dir}
				if attr, ok := b.Body.Attributes["source"]; ok {
					if val, diags := attr.Expr.Value(nil); diags == nil || !diags.HasErrors() {
						if val.Type() == cty.String {
							if child, ok := resolveModuleSource(root, dir, val.AsString()); ok {
								call.childDir = child
								moduleDirSet[child] = struct{}{}
							}
						}
					}
				}

				if attr, ok := b.Body.Attributes["providers"]; ok {
					if obj, ok := attr.Expr.(*hclsyntax.ObjectConsExpr); ok {
						call.providers = map[string]string{}
						for _, item := range obj.Items {
							key := objectKeyToString(item.KeyExpr)
							if key == "" {
								continue
							}
							if ref := providerRefFromExpr(item.ValueExpr); ref != "" {
								call.providers[key] = ref
								resolver.recordUsage(ref)
								resolver.ensureAlias(key)
							}
						}
					}
				}
				calls = append(calls, call)

			case "terraform":
				aliases := terraformRequiredAliases(b)
				if len(aliases) > 0 {
					scope := resolver.scope(dir)
					for _, alias := range aliases {
						scope.allowAlias(alias)
						resolver.ensureAlias(alias)
					}
				}
			}
		}
	}

	resolver.moduleDirs = make([]string, 0, len(moduleDirSet))
	for dir := range moduleDirSet {
		resolver.moduleDirs = append(resolver.moduleDirs, dir)
	}
	sort.Slice(resolver.moduleDirs, func(i, j int) bool {
		return len(resolver.moduleDirs[i]) > len(resolver.moduleDirs[j])
	})

	if _, ok := resolver.scopes[root]; !ok {
		resolver.scopes[root] = newProviderScope(root)
	}
	for _, dir := range resolver.moduleDirs {
		if _, ok := resolver.scopes[dir]; !ok {
			resolver.scopes[dir] = newProviderScope(dir)
		}
	}

	for _, def := range defs {
		scopePath := resolver.scopeForDir(def.dir)
		resolver.scope(scopePath).addAlias(def.alias, true)
	}

	for _, call := range calls {
		if call.childDir == "" {
			continue
		}
		scope := resolver.scope(call.childDir)
		for alias := range call.providers {
			scope.addAlias(alias, false)
			resolver.ensureAlias(alias)
		}
	}

	return resolver, nil
}

func newProviderScope(path string) *providerScope {
	return &providerScope{
		Path:    path,
		Aliases: map[string]*providerAlias{},
		Allowed: map[string]struct{}{},
	}
}

func (r *providerResolver) ensureAlias(name string) {
	if name == "" {
		return
	}
	if _, ok := r.aliasUsage[name]; !ok {
		r.aliasUsage[name] = 0
	}
}

func (r *providerResolver) recordUsage(name string) {
	if name == "" {
		return
	}
	r.aliasUsage[name]++
	r.ensureAlias(name)
}

func (r *providerResolver) scope(path string) *providerScope {
	if s, ok := r.scopes[path]; ok {
		return s
	}
	s := newProviderScope(path)
	r.scopes[path] = s
	return s
}

func (r *providerResolver) scopeForDir(dir string) string {
	dir = filepath.Clean(dir)
	for _, mod := range r.moduleDirs {
		if dir == mod || strings.HasPrefix(dir, mod+string(os.PathSeparator)) {
			return mod
		}
	}
	return r.root
}

func (s *providerScope) addAlias(name string, defined bool) *providerAlias {
	if name == "" {
		return nil
	}
	if entry, ok := s.Aliases[name]; ok {
		if defined {
			entry.Defined = true
		}
		return entry
	}
	entry := &providerAlias{
		Name:    name,
		Type:    providerTypeFromAlias(name),
		Defined: defined,
	}
	s.Aliases[name] = entry
	return entry
}

func (s *providerScope) allowAlias(name string) {
	if name == "" {
		return
	}
	if s.Allowed == nil {
		s.Allowed = map[string]struct{}{}
	}
	s.Allowed[name] = struct{}{}
	s.addAlias(name, false)
}

func (r *providerResolver) resolve(dir, preferredType string) (providerSelection, error) {
	scopePath := r.scopeForDir(dir)
	scope := r.scope(scopePath)

	candidates := scope.collectAliases()
	if len(candidates) == 0 {
		if len(scope.Allowed) > 0 || r.hasDefs {
			return providerSelection{}, fmt.Errorf("%w in %s", errNoProviderInScope, scopePath)
		}
		return providerSelection{}, errNoProviderDefinitions
	}

	filtered := filterAliasesByType(candidates, preferredType)
	if len(filtered) > 0 {
		candidates = filtered
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		ci := r.aliasUsage[candidates[i].Name]
		cj := r.aliasUsage[candidates[j].Name]
		if ci == cj {
			return candidates[i].Name < candidates[j].Name
		}
		return ci > cj
	})

	best := candidates[0]
	return providerSelection{Alias: best.Name, Type: best.Type}, nil
}

func buildProviderAliasName(typ, alias string) string {
	if alias == "" {
		return typ
	}
	return fmt.Sprintf("%s.%s", typ, alias)
}

func (r *providerResolver) requiredAliasesForModule(filePath string, block *hclsyntax.Block) []string {
	source, ok := moduleSourceString(block)
	if !ok {
		return nil
	}
	parentDir := filepath.Dir(filePath)
	childDir, ok := resolveModuleSource(r.root, parentDir, source)
	if !ok {
		return nil
	}
	scope := r.scope(childDir)
	return scope.allowedAliasList()
}

func (s *providerScope) collectAliases() []*providerAlias {
	if len(s.Allowed) > 0 {
		var out []*providerAlias
		for name := range s.Allowed {
			if alias, ok := s.Aliases[name]; ok {
				out = append(out, alias)
			}
		}
		return out
	}
	out := make([]*providerAlias, 0, len(s.Aliases))
	for _, alias := range s.Aliases {
		out = append(out, alias)
	}
	return out
}

func (s *providerScope) allowedAliasList() []string {
	if len(s.Allowed) == 0 {
		return nil
	}
	list := make([]string, 0, len(s.Allowed))
	for name := range s.Allowed {
		if _, ok := s.Aliases[name]; ok {
			list = append(list, name)
		}
	}
	sort.Strings(list)
	return list
}

func filterAliasesByType(list []*providerAlias, typ string) []*providerAlias {
	if typ == "" {
		return nil
	}
	var out []*providerAlias
	for _, alias := range list {
		if alias.Type == typ {
			out = append(out, alias)
		}
	}
	return out
}

func providerTypeFromAlias(alias string) string {
	if alias == "" {
		return ""
	}
	if idx := strings.IndexRune(alias, '.'); idx >= 0 {
		return alias[:idx]
	}
	return alias
}

func resolveModuleSource(root, parentDir, source string) (string, bool) {
	if source == "" {
		return "", false
	}

	if strings.Contains(source, "::") {
		return "", false
	}

	var target string
	if filepath.IsAbs(source) {
		target = source
	} else {
		target = filepath.Join(parentDir, source)
	}
	target = filepath.Clean(target)

	if !strings.HasPrefix(target, root) {
		return "", false
	}
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return target, true
}

func objectKeyToString(expr hclsyntax.Expression) string {
	if expr == nil {
		return ""
	}
	if ref := providerRefFromExpr(expr); ref != "" {
		return ref
	}
	if val, diags := expr.Value(nil); diags == nil || !diags.HasErrors() {
		if val.Type() == cty.String {
			return val.AsString()
		}
	}
	switch e := expr.(type) {
	case *hclsyntax.TemplateExpr:
		if len(e.Parts) == 1 {
			if wrap, ok := e.Parts[0].(*hclsyntax.TemplateWrapExpr); ok {
				return objectKeyToString(wrap.Wrapped)
			}
		}
	case *hclsyntax.ScopeTraversalExpr:
		return traversalToProviderRef(e.Traversal)
	case *hclsyntax.RelativeTraversalExpr:
		return traversalToProviderRef(e.Traversal)
	case *hclsyntax.ObjectConsKeyExpr:
		return objectKeyToString(e.Wrapped)
	}
	return ""
}

func terraformRequiredAliases(block *hclsyntax.Block) []string {
	if block.Type != "terraform" {
		return nil
	}
	var out []string
	for _, child := range block.Body.Blocks {
		if child.Type != "required_providers" {
			continue
		}
		for _, attr := range child.Body.Attributes {
			out = append(out, configurationAliases(attr)...)
		}
	}
	return out
}

func configurationAliases(attr *hclsyntax.Attribute) []string {
	obj, ok := attr.Expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return nil
	}
	for _, item := range obj.Items {
		key := objectKeyToString(item.KeyExpr)
		if key != "configuration_aliases" {
			continue
		}
		return aliasListFromExpr(item.ValueExpr)
	}
	return nil
}

func aliasListFromExpr(expr hclsyntax.Expression) []string {
	switch v := expr.(type) {
	case *hclsyntax.TupleConsExpr:
		var list []string
		for _, e := range v.Exprs {
			if ref := providerRefFromExpr(e); ref != "" {
				list = append(list, ref)
				continue
			}
			if val, diags := e.Value(nil); diags == nil || !diags.HasErrors() {
				if val.Type() == cty.String {
					list = append(list, val.AsString())
				}
			}
		}
		return list
	default:
		if ref := providerRefFromExpr(expr); ref != "" {
			return []string{ref}
		}
		if val, diags := expr.Value(nil); diags == nil || !diags.HasErrors() {
			if val.Type() == cty.String {
				return []string{val.AsString()}
			}
		}
	}
	return nil
}

func moduleSourceString(block *hclsyntax.Block) (string, bool) {
	attr, ok := block.Body.Attributes["source"]
	if !ok {
		return "", false
	}
	val, diags := attr.Expr.Value(nil)
	if diags != nil && diags.HasErrors() {
		return "", false
	}
	if val.Type() != cty.String {
		return "", false
	}
	return val.AsString(), true
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

func ScanFileAfterFix(path string, cfg *config.Config) ([]model.Finding, error) {
	return parser.ParseFile(path, cfg)
}

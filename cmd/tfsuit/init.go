package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	hcl "github.com/hashicorp/hcl/v2"
	hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/josdagaro/tfsuit/internal/parser"
)

// newInitCmd scaffolds a starter tfsuit.hcl by inspecting current labels
func newInitCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "init [path]",
		Short: "Interactively generate an initial tfsuit.hcl",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			return runInit(root)
		},
	}
	return c
}

type counts struct {
	namesByKind map[string][]string // kind -> names
	hasUpper    bool
	hasHyphen   bool
}

func runInit(root string) error {
	files, err := parser.Discover(root)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		fmt.Println("No Terraform files found (\".tf\"). Nothing to infer.")
		return nil
	}

	// Parse labels
	cts := &counts{namesByKind: map[string][]string{}}
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			continue
		}
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
				name := b.Labels[0]
				cts.bump(b.Type, name)
			case "resource":
				if len(b.Labels) < 2 {
					continue
				}
				name := b.Labels[1]
				cts.bump("resource", name)
			}
		}
	}

	// Interactive Q&A
	fmt.Printf("Found %d files. Labels: variables=%d, outputs=%d, modules=%d, resources=%d\n",
		len(files), len(cts.namesByKind["variable"]), len(cts.namesByKind["output"]), len(cts.namesByKind["module"]), len(cts.namesByKind["resource"]))

	reader := bufio.NewReader(os.Stdin)

	allowUpper := askYesNo(reader, fmt.Sprintf("Allow uppercase letters? detected=%v [y/N]: ", cts.hasUpper), false)
	allowHyphen := askYesNo(reader, fmt.Sprintf("Allow hyphens '-'? detected=%v [y/N]: ", cts.hasHyphen), false)

	// Module suffix heuristic: if many modules end with _<letters>, suggest optional suffix
	suggestSuffix := suggestModuleSuffix(cts.namesByKind["module"]) > 0
	useModuleSuffix := askYesNo(reader, fmt.Sprintf("Enforce optional module suffix (_[a-z]+)? suggested=%v [y/N]: ", suggestSuffix), suggestSuffix)

	// Common ignores: pick top frequent names for variables/resources
	candIgnores := topFrequent(append(cts.namesByKind["variable"], cts.namesByKind["resource"]...), 5)
	var ignoreExact []string
	if len(candIgnores) > 0 {
		fmt.Printf("Suggested ignore_exact (comma separated, Enter to accept all, '-' for none): %s\n", strings.Join(candIgnores, ", "))
		fmt.Print("Your choice: ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		switch {
		case line == "":
			ignoreExact = candIgnores
		case line == "-":
			// none
		default:
			// user provided
			parts := strings.Split(line, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					ignoreExact = append(ignoreExact, p)
				}
			}
		}
	}
	addExperimental := askYesNo(reader, "Add ignore_regex entry for '.*experimental.*'? [y/N]: ", false)

	// Compose patterns
	base := charClass(allowUpper, allowHyphen)
	varPattern := fmt.Sprintf("^[%s]+$", base)
	outPattern := varPattern
	modPattern := varPattern
	if useModuleSuffix {
		modPattern = fmt.Sprintf("^[%s]+(_[a-z]+)?$", base)
	}
	resPattern := varPattern

	// Render file
	content := renderConfig(varPattern, outPattern, modPattern, resPattern, ignoreExact, addExperimental)

	outPath := filepath.Join(root, "tfsuit.hcl")
	if _, err := os.Stat(outPath); err == nil {
		if !askYesNo(reader, fmt.Sprintf("%s exists. Overwrite? [y/N]: ", outPath), false) {
			fmt.Println("Aborted.")
			return nil
		}
	}
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n", outPath)
	return nil
}

func (c *counts) bump(kind, name string) {
	c.namesByKind[kind] = append(c.namesByKind[kind], name)
	for _, ch := range name {
		if ch >= 'A' && ch <= 'Z' {
			c.hasUpper = true
		}
		if ch == '-' {
			c.hasHyphen = true
		}
	}
}

func askYesNo(r *bufio.Reader, prompt string, def bool) bool {
	fmt.Print(prompt)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return def
	}
	if line == "y" || line == "yes" {
		return true
	}
	if line == "n" || line == "no" {
		return false
	}
	// fallback to default
	return def
}

func charClass(upper, hyphen bool) string {
	base := "a-z0-9_"
	if upper {
		base = "A-Za-z0-9_"
	}
	if hyphen {
		base += "-"
	}
	return base
}

func suggestModuleSuffix(mods []string) int {
	// Count modules ending with _[a-z]+
	re := regexp.MustCompile(`_[a-z]+$`)
	n := 0
	for _, m := range mods {
		if re.MatchString(m) {
			n++
		}
	}
	return n
}

func topFrequent(items []string, maxN int) []string {
	if len(items) == 0 {
		return nil
	}
	freq := map[string]int{}
	for _, it := range items {
		freq[it]++
	}
	type kv struct {
		k string
		v int
	}
	var list []kv
	for k, v := range freq {
		list = append(list, kv{k, v})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].v == list[j].v {
			return list[i].k < list[j].k
		}
		return list[i].v > list[j].v
	})
	out := make([]string, 0, maxN)
	for i := 0; i < len(list) && i < maxN; i++ {
		out = append(out, list[i].k)
	}
	return out
}

func renderConfig(varPattern, outPattern, modPattern, resPattern string, ignoreExact []string, addExperimental bool) string {
	var b strings.Builder
	b.WriteString("# tfsuit.hcl — generated by `tfsuit init`\n")
	b.WriteString("# Tune the patterns below to your org’s conventions.\n\n")

	// variables
	b.WriteString("variables {\n")
	b.WriteString(fmt.Sprintf("  pattern      = \"%s\"\n", varPattern))
	if len(ignoreExact) > 0 {
		b.WriteString(fmt.Sprintf("  ignore_exact = [%s]\n", quoteCSV(ignoreExact)))
	}
	b.WriteString("}\n\n")

	// outputs
	b.WriteString("outputs {\n")
	b.WriteString(fmt.Sprintf("  pattern = \"%s\"\n", outPattern))
	b.WriteString("}\n\n")

	// modules
	b.WriteString("modules {\n")
	b.WriteString(fmt.Sprintf("  pattern = \"%s\"\n", modPattern))
	if addExperimental {
		b.WriteString("  ignore_regex = [\".*experimental.*\"]\n")
	}
	b.WriteString("}\n\n")

	// resources
	b.WriteString("resources {\n")
	b.WriteString(fmt.Sprintf("  pattern = \"%s\"\n", resPattern))
	b.WriteString("}\n")

	return b.String()
}

func quoteCSV(items []string) string {
	q := make([]string, 0, len(items))
	for _, s := range items {
		q = append(q, fmt.Sprintf("\"%s\"", s))
	}
	return strings.Join(q, ", ")
}

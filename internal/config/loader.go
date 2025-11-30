package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"
)

type Rule struct {
	Pattern         string   `hcl:"pattern" json:"pattern"`
	IgnoreExact     []string `hcl:"ignore_exact,optional" json:"ignore_exact"`
	IgnoreRegex     []string `hcl:"ignore_regex,optional" json:"ignore_regex"`
	RequireProvider *bool    `hcl:"require_provider,optional" json:"require_provider,omitempty"`

	patternRe    *regexp.Regexp
	ignoreReList []*regexp.Regexp
	requireProv  bool
}

type Config struct {
	Variables Rule  `hcl:"variables,block" json:"variables"`
	Outputs   Rule  `hcl:"outputs,block" json:"outputs"`
	Modules   Rule  `hcl:"modules,block" json:"modules"`
	Resources Rule  `hcl:"resources,block" json:"resources"`
	Data      *Rule `hcl:"data,block" json:"data,omitempty"`
}

func (r *Rule) compile() error {
	re, err := regexp.Compile(r.Pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern '%s': %w", r.Pattern, err)
	}
	r.patternRe = re

	for _, ig := range r.IgnoreRegex {
		igr, err := regexp.Compile(ig)
		if err != nil {
			return fmt.Errorf("invalid ignore_regex '%s': %w", ig, err)
		}
		r.ignoreReList = append(r.ignoreReList, igr)
	}
	return nil
}

func (r *Rule) Matches(name string) bool {
	return r.patternRe.MatchString(name)
}

func (r *Rule) IsIgnored(name string) bool {
	for _, ex := range r.IgnoreExact {
		if name == ex {
			return true
		}
	}
	for _, re := range r.ignoreReList {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

func (r *Rule) setRequireProvider(defaultVal bool) {
	if r.RequireProvider != nil {
		r.requireProv = *r.RequireProvider
		return
	}
	r.requireProv = defaultVal
}

func (r *Rule) RequiresProvider() bool {
	return r.requireProv
}

func (c *Config) compileRules() error {
	if c.Data == nil {
		c.Data = &Rule{Pattern: ".*"}
	} else if c.Data.Pattern == "" {
		c.Data.Pattern = ".*"
	}

	type ruleDef struct {
		rule *Rule
		def  bool
	}

	rules := []ruleDef{
		{rule: &c.Variables, def: false},
		{rule: &c.Outputs, def: false},
		{rule: &c.Modules, def: true},
		{rule: &c.Resources, def: false},
		{rule: c.Data, def: false},
	}

	for _, rd := range rules {
		if err := rd.rule.compile(); err != nil {
			return err
		}
		rd.rule.setRequireProvider(rd.def)
	}
	return nil
}

// Load reads and parses a HCL or JSON config file.
func Load(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config

	switch filepath.Ext(path) {
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	default: // assume HCL
		file, diags := hclsyntax.ParseConfig(data, path, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			return nil, fmt.Errorf("%s", diags.Error())
		}
		if diags := gohcl.DecodeBody(file.Body, nil, &cfg); diags.HasErrors() {
			return nil, fmt.Errorf("%s", diags.Error())
		}
	}

	if err := cfg.compileRules(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

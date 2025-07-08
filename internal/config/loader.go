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
    Pattern     string   `hcl:"pattern" json:"pattern"`
    IgnoreExact []string `hcl:"ignore_exact,optional" json:"ignore_exact"`
    IgnoreRegex []string `hcl:"ignore_regex,optional" json:"ignore_regex"`

    patternRe    *regexp.Regexp
    ignoreReList []*regexp.Regexp
}

type Config struct {
    Variables Rule `hcl:"variables,block" json:"variables"`
    Outputs   Rule `hcl:"outputs,block" json:"outputs"`
    Modules   Rule `hcl:"modules,block" json:"modules"`
    Resources Rule `hcl:"resources,block" json:"resources"`
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

func (c *Config) compileRules() error {
    rules := []*Rule{&c.Variables, &c.Outputs, &c.Modules, &c.Resources}
    for _, r := range rules {
        if err := r.compile(); err != nil {
            return err
        }
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

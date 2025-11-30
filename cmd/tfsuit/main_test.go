package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoPath(parts ...string) string {
	all := append([]string{"..", ".."}, parts...)
	return filepath.Join(all...)
}

func TestRunScanSuccessAndFailFlag(t *testing.T) {
	cfgFile = repoPath("samples", "simple", "tfsuit.hcl")
	format = "json"
	fail = false
	if err := runScan(repoPath("samples", "simple")); err != nil {
		t.Fatalf("runScan success: %v", err)
	}

	fail = true
	if err := runScan(repoPath("samples", "simple")); err == nil {
		t.Fatalf("expected error when --fail flag enabled")
	}
	fail = false
}

func TestFixCommandRespectsConfigFlag(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "tfsuit.hcl")
	if err := os.WriteFile(cfgPath, []byte(`
variables { pattern = "^[a-z0-9_]+$" }
outputs   { pattern = "^[a-z0-9_]+$" }
modules   { pattern = "^[a-z0-9_]+$" }
resources { pattern = "^[a-z0-9_]+$" }`), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	tfPath := filepath.Join(dir, "bad.tf")
	if err := os.WriteFile(tfPath, []byte(`module "Bad-Name" { source = "../" }`), 0o644); err != nil {
		t.Fatalf("write tf: %v", err)
	}
	providersPath := filepath.Join(dir, "providers.tf")
	if err := os.WriteFile(providersPath, []byte(`provider "aws" {
  alias = "primary"
}`), 0o644); err != nil {
		t.Fatalf("write providers: %v", err)
	}

	cmd := newFixCmd()
	cmd.SetArgs([]string{dir, "-c", cfgPath})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("fix command execute: %v", err)
	}
}

func TestRunInitCreatesConfig(t *testing.T) {
	dir := t.TempDir()
	tfPath := filepath.Join(dir, "main.tf")
	if err := os.WriteFile(tfPath, []byte(`module "demo" { source = "../" }`), 0o644); err != nil {
		t.Fatalf("write tf: %v", err)
	}

	inputFile := filepath.Join(dir, "answers.txt")
	if err := os.WriteFile(inputFile, []byte("\n\n\n\n"), 0o644); err != nil {
		t.Fatalf("write answers: %v", err)
	}
	f, err := os.Open(inputFile)
	if err != nil {
		t.Fatalf("open answers: %v", err)
	}
	defer f.Close()

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()
	os.Stdin = f

	if err := runInit(dir); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	out, err := os.ReadFile(filepath.Join(dir, "tfsuit.hcl"))
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}
	if !strings.Contains(string(out), "variables") {
		t.Fatalf("generated config missing content:\n%s", out)
	}
}

func TestInitCommand(t *testing.T) {
	dir := t.TempDir()
	tfPath := filepath.Join(dir, "main.tf")
	if err := os.WriteFile(tfPath, []byte(`module "demo" { source = "../" }`), 0o644); err != nil {
		t.Fatalf("write tf: %v", err)
	}

	inputFile := filepath.Join(dir, "answers.txt")
	if err := os.WriteFile(inputFile, []byte("\n\n\n\n"), 0o644); err != nil {
		t.Fatalf("write answers: %v", err)
	}
	f, err := os.Open(inputFile)
	if err != nil {
		t.Fatalf("open answers: %v", err)
	}
	defer f.Close()

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()
	os.Stdin = f

	cmd := newInitCmd()
	cmd.SetArgs([]string{dir})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command execute: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "tfsuit.hcl")); err != nil {
		t.Fatalf("expected config file: %v", err)
	}
}

func TestRootScanCommand(t *testing.T) {
	cmd := rootCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{
		"scan",
		repoPath("samples", "simple"),
		"-c", repoPath("samples", "simple", "tfsuit.hcl"),
		"-f", "json",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("root scan command: %v", err)
	}
}

func TestInitHelpersCoverages(t *testing.T) {
	if got := charClass(true, true); !strings.Contains(got, "A-Z") || !strings.Contains(got, "-") {
		t.Fatalf("charClass unexpected: %s", got)
	}
	mods := []string{"app_qa", "db", "app_stage"}
	if suggestModuleSuffix(mods) == 0 {
		t.Fatalf("expected suffix suggestion for %v", mods)
	}
	items := []string{"foo", "bar", "foo", "baz"}
	top := topFrequent(items, 2)
	if len(top) != 2 || top[0] != "foo" {
		t.Fatalf("topFrequent mismatch: %v", top)
	}
	configText := renderConfig("a", "b", "c", "d", "e", []string{"foo", "bar"}, true)
	if !strings.Contains(configText, "ignore_exact") || !strings.Contains(configText, "ignore_regex") {
		t.Fatalf("renderConfig missing expected sections:\n%s", configText)
	}
	if got := quoteCSV([]string{"a", "b"}); got != `"a", "b"` {
		t.Fatalf("quoteCSV output mismatch: %s", got)
	}

	reader := bufio.NewReader(strings.NewReader("y\n"))
	if !askYesNo(reader, "", false) {
		t.Fatalf("askYesNo should return true for yes")
	}
	reader = bufio.NewReader(strings.NewReader("n\n"))
	if askYesNo(reader, "", true) {
		t.Fatalf("askYesNo should return false for no")
	}
	reader = bufio.NewReader(strings.NewReader("\n"))
	if !askYesNo(reader, "", true) {
		t.Fatalf("askYesNo should use default when empty")
	}
}

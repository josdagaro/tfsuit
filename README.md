# tfsuit

<img src="assets/logo/tfsuit-logo.png" alt="tfsuit-logo" width="200"/>

[![Release](https://github.com/josdagaro/tfsuit/actions/workflows/release.yml/badge.svg?branch=main)](https://github.com/josdagaro/tfsuit/actions/workflows/release.yml) [![tfsuit CI](https://github.com/josdagaro/tfsuit/actions/workflows/ci.yml/badge.svg)](https://github.com/josdagaro/tfsuit/actions/workflows/ci.yml) [![tfsuit scan](https://github.com/josdagaro/tfsuit/actions/workflows/tfsuit.yml/badge.svg)](https://github.com/josdagaro/tfsuit/actions/workflows/tfsuit.yml)

> **Fast, opinionated Terraform naming linter & fixer – written in Go**

`tfsuit` helps you enforce consistent, organisation‑wide naming rules for every Terraform variable, output, module and resource – in your editor, in CI and in your pull‑requests.

---

## 🎯 Why tfsuit?

- Enforce naming policies once and share them across repos, teams and CI
- Catch inconsistent Terraform labels before they reach review or production
- Auto‑fix issues while keeping cross‑references in sync (no manual renames)
- Integrates with GitHub Actions, SARIF code scanning and editor tooling

---

## ✨ Key features (v1)

|                        | Feature                                                  | Notes                                         |
| ---------------------- | -------------------------------------------------------- | --------------------------------------------- |
| **Ultra‑fast core**    | Go implementation ▶ multi‑CPU parsing, intelligent cache | 10‑50× faster than the original Bash version  |
| **Configurable rules** | HCL or JSON (`tfsuit.hcl`)                               | Per‑type patterns, allow‑lists / ignore‑regex |
| **Linter modes**       | `scan` (read‑only)                                       | Pretty, JSON or SARIF output                  |
| **Auto‑fixer**         | `fix` – rewrites labels, updates all cross‑references    | `--dry-run` to preview, `--write` to apply    |
| **Code Scanning**      | SARIF + GitHub annotations                               | PR checklist + summary comment                |
| **GitHub Action**      | `uses: josdagaro/tfsuit/action@v3`                       | Runs in Docker, no build step                 |
| **Homebrew formula**   | `brew install josdagaro/tfsuit/tfsuit`                   | macOS / Linux                                 |
| **Docker image**       | `ghcr.io/josdagaro/tfsuit:<tag>`                         | Static binary, 6 MiB                          |
| **VS Code extension**  | *Preview*: LSP‑based inline diagnostics & quick‑fix      | Coming soon                                   |

---

## ⚡ Quick start

```bash
# 1. Install
brew install josdagaro/tfsuit/tfsuit

# 2. Drop a config file in your repo root
cat <<'EOF' > tfsuit.hcl
variables { pattern = "^[a-z0-9_]+$" }
resources { pattern = "^[a-z0-9_]+$" }
EOF

# 3. Scan your Terraform project
tfsuit scan ./infra
```

`tfsuit` exits non‑zero when violations are found, so you can wire it directly into CI.

---

## 🚀 Installation

### Homebrew (macOS/Linux)

```bash
brew tap josdagaro/tfsuit
brew install tfsuit
```

Update to the latest tagged release:

```bash
brew update
brew upgrade tfsuit
```

Validate your installation:

```bash
tfsuit --version
```

### Binary release

Grab the archive for your OS from the [GitHub Releases](https://github.com/josdagaro/tfsuit/releases) page, extract and move `tfsuit` to a directory on your `$PATH`.

### Docker

```bash
# latest stable
docker run --rm -v "$PWD:/src" ghcr.io/josdagaro/tfsuit:latest scan /src
```

### GitHub Action

Add to your workflow:

```yaml
- uses: josdagaro/tfsuit/action@v1
  with:
    path: ./infra                # directory to scan (default '.')
    config: .github/tfsuit.hcl   # your rule file (default 'tfsuit.hcl')
    format: sarif                # pretty | json | sarif
    fail: true                   # fail the job if violations found
```

The action automatically uploads the SARIF file to GitHub Code Scanning.

---

## 📑 Configuration (`tfsuit.hcl`)

```hcl
variables {
  pattern      = "^[a-z0-9_]+$"
  ignore_exact = ["aws_region"]
}

outputs {
  pattern = "^[a-z0-9_]+$"
}

modules {
  pattern      = "^[a-z0-9_]+(_[a-z]+)?$"
  ignore_regex = [".*experimental.*"]
}

resources {
  pattern = "^[a-z0-9_]+$"
}
```

*Compile‑time validation* – invalid regex is caught at startup.

---

## 🔍 CLI usage

```bash
tfsuit scan [path]           # lint only
  -c, --config <file>        # config file
  -f, --format pretty|json|sarif
      --fail                 # exit 1 on violations

tfsuit fix [path]            # auto‑fix labels
      --dry-run              # show diff
      --write                # apply changes
```

Example:

```bash
# CI – fail if naming is wrong and upload SARIF
mkdir results
tfsuit scan ./infra --fail --format sarif > results/tfsuit.sarif
```

---

## 🧪 Examples

Given a Terraform resource with a non‑conforming label:

```hcl
resource "aws_s3_bucket" "BadBucket" {
  bucket = "example"
}
```

Scan output (pretty format):

```text
$ tfsuit scan ./infra
infra/main.tf:2:15  resource.aws_s3_bucket.BadBucket  label "BadBucket" does not match "^[a-z0-9_]+$"
```

Auto‑fix and review the change:

```bash
tfsuit fix ./infra --dry-run   # see proposed rename
tfsuit fix ./infra --write     # apply updates to all references
```

The fixer rewrites references (modules, locals, outputs) to keep your code compiling.

---

## 🧩 VS Code (preview)

The upcoming extension provides live diagnostics and `Quick Fix…` to rename variables safely. Watch the [project board](https://github.com/josdagaro/tfsuit/projects/1) for progress.

---

## 🛠 Development

```bash
make test        # go vet + unit tests
make snapshot    # local goreleaser build
```

### Prerequisites

- Go 1.21+ (matches the version in `go.mod`)
- `make`
- (optional) [GoReleaser](https://goreleaser.com) for snapshot packaging

### Build & run locally

```bash
go build ./cmd/tfsuit    # compile binary into current directory
./tfsuit --help          # inspect available commands

go run ./cmd/tfsuit scan ./testdata/terraform
```

### Test before pushing

```bash
go test ./...
make test                # wraps gofmt, go vet, go test
```

Run the fixer against fixtures to verify behaviour:

```bash
go run ./cmd/tfsuit fix ./internal/parser/testdata --dry-run
```

### GoReleaser dry runs

Use snapshot releases to emulate the CI pipeline without publishing artifacts:

```bash
make snapshot            # goreleaser release --snapshot --clean
```

The command builds platform packages, Docker image and the Homebrew formula locally so you can spot issues before opening a release PR.

### Release pipeline

- **SemVer** determined from PR label (`major` / `minor` / `patch`).
- **GoReleaser** builds binaries, Docker image, Homebrew formula.
- Tags `vX.Y.Z`, moving tags `vX`, `vX.Y`.

Details in `.github/workflows/release.yml`.

---

## 📜 License

MIT License – see [LICENSE](LICENSE).

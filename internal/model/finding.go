package model

// Finding represents a naming‑rule violation discovered in a .tf file.
// Other packages (engine, parser, reporters) depend on this model, avoiding
// cyclical imports.

type Finding struct {
    File    string `json:"file"` // path relative to repo root
    Line    int    `json:"line"` // 1‑based line number
    Kind    string `json:"kind"` // variable|output|resource|module
    Name    string `json:"name"` // offending identifier
    Message string `json:"message"` // human‑readable description
}

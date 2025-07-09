package model

type Finding struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

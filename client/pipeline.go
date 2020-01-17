package client

import (
	"strings"
)

type Pipeline struct {
	op []string
}

func (p *Pipeline) Canonical() string {
	return strings.Join(p.op, " | ")
}

func (p *Pipeline) String() string {
	return strings.Join(p.op, "\n")
}

func NewPipeline(s string) *Pipeline {
	var ops []string
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		for _, stmt := range strings.Split(line, "|") {
			if op := strings.TrimSpace(stmt); op != "" && !strings.HasPrefix(op, "//") {
				ops = append(ops, op)
			}
		}
	}
	return &Pipeline{ops}
}

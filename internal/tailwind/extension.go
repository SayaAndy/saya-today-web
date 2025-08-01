package tailwind

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"
)

type TailwindExtension struct{}

func NewTailwindExtension() goldmark.Extender {
	return &TailwindExtension{}
}

func (e *TailwindExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&TailwindTransformer{}, 500),
		),
	)
}

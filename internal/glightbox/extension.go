package glightbox

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Extension that combines parser and renderer
type GLightboxExtension struct{}

func NewGLightboxExtension() goldmark.Extender {
	return &GLightboxExtension{}
}

func (e *GLightboxExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithBlockParsers(
			util.Prioritized(NewGLightboxParser(), 500),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(NewGLightboxHTMLRenderer(), 500),
		),
	)
}

package lightgallery

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Extension that combines parser and renderer
type LightGalleryExtension struct{}

func NewLightGalleryExtension() goldmark.Extender {
	return &LightGalleryExtension{}
}

func (e *LightGalleryExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithBlockParsers(
			util.Prioritized(NewLightGalleryParser(), 500),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(NewLightGalleryHTMLRenderer(), 500),
		),
	)
}

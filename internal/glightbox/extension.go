package glightbox

import (
	"github.com/SayaAndy/saya-today-web/config"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Extension that combines parser and renderer
type GLightboxExtension struct {
	photoStorage config.PhotoStorageConfig
}

func NewGLightboxExtension(photoStorage config.PhotoStorageConfig) goldmark.Extender {
	return &GLightboxExtension{photoStorage}
}

func (e *GLightboxExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithBlockParsers(
			util.Prioritized(NewGLightboxParser(), 500),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(NewGLightboxHTMLRenderer(e.photoStorage), 500),
		),
	)
}

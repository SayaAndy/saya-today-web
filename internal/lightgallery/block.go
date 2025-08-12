package lightgallery

import (
	"time"

	"github.com/yuin/goldmark/ast"
)

// LightGalleryBlock represents a light gallery block in the AST
type LightGalleryBlock struct {
	ast.BaseBlock
	Images   []LightGalleryImage
	Location *time.Location
}

type LightGalleryImage struct {
	URL     string
	Caption string
}

var KindLightGalleryBlock = ast.NewNodeKind("LightGalleryBlock")

// Dump implements ast.Node.Dump
func (n *LightGalleryBlock) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// Kind implements ast.Node.Kind
func (n *LightGalleryBlock) Kind() ast.NodeKind {
	return KindLightGalleryBlock
}

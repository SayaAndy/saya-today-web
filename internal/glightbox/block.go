package glightbox

import (
	"time"

	"github.com/yuin/goldmark/ast"
)

// GLightboxBlock represents a light gallery block in the AST
type GLightboxBlock struct {
	ast.BaseBlock
	Images   []GLightboxImage
	Location *time.Location
}

type GLightboxImage struct {
	URL     string
	Caption []byte
}

var KindGLightboxBlock = ast.NewNodeKind("GLightboxBlock")

// Dump implements ast.Node.Dump
func (n *GLightboxBlock) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// Kind implements ast.Node.Kind
func (n *GLightboxBlock) Kind() ast.NodeKind {
	return KindGLightboxBlock
}

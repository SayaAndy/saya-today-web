package lightgallery

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type LightGalleryParser struct{}

func NewLightGalleryParser() parser.BlockParser {
	return &LightGalleryParser{}
}

func (p *LightGalleryParser) Trigger() []byte {
	return []byte{'{'}
}

func (p *LightGalleryParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()

	if !bytes.HasPrefix(line, []byte("{Gallery}")) {
		return nil, parser.NoChildren
	}

	trimmed := bytes.TrimSpace(line)
	if !bytes.Equal(trimmed, []byte("{Gallery}")) {
		return nil, parser.NoChildren
	}

	return &LightGalleryBlock{}, parser.NoChildren
}

func (p *LightGalleryParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	if len(line) == 0 || segment.Len() == 0 {
		return parser.Close
	}

	trimmed := bytes.TrimSpace(line)
	if bytes.Equal(trimmed, []byte("{Gallery}")) {
		reader.AdvanceLine()
		return parser.Close
	}

	gallery := node.(*LightGalleryBlock)
	lineStr := string(trimmed)

	parts := strings.SplitN(lineStr, "|", 2)
	url := strings.TrimSpace(parts[0])
	caption := ""

	if len(parts) > 1 {
		caption = strings.TrimSpace(parts[1])
	}

	gallery.Images = append(gallery.Images, LightGalleryImage{
		URL:     url,
		Caption: caption,
	})

	return parser.Continue | parser.NoChildren
}

func (p *LightGalleryParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
}

func (p *LightGalleryParser) CanInterruptParagraph() bool {
	return true
}

func (p *LightGalleryParser) CanAcceptIndentedLine() bool {
	return false
}

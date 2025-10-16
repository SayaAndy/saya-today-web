package glightbox

import (
	"bytes"
	"log/slog"
	"regexp"
	"time"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type GLightboxParser struct{}

func NewGLightboxParser() parser.BlockParser {
	return &GLightboxParser{}
}

func (p *GLightboxParser) Trigger() []byte {
	return []byte{'{'}
}

func (p *GLightboxParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()

	if !bytes.HasPrefix(line, []byte("{Gallery:")) {
		return nil, parser.NoChildren
	}

	r := regexp.MustCompile(`^\{Gallery:([A-Za-z0-9\+\-/]+)\}$`)

	trimmed := bytes.TrimSpace(line)
	parts := r.FindSubmatch(trimmed)
	if len(parts) < 2 {
		slog.Warn("invalid gallery header format", slog.String("line", string(trimmed)), slog.Int("submatch_count", len(parts)))
		return nil, parser.NoChildren
	}

	loc, err := time.LoadLocation(string(parts[1]))
	if err != nil {
		slog.Warn("invalid location specified for a gallery", slog.String("error", err.Error()), slog.String("line", string(trimmed)), slog.String("location", string(parts[1])))
		return nil, parser.NoChildren
	}

	return &GLightboxBlock{Location: loc}, parser.NoChildren
}

func (p *GLightboxParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	if len(line) == 0 || segment.Len() == 0 {
		return parser.Continue | parser.NoChildren
	}

	trimmed := bytes.TrimSpace(line)
	if bytes.Equal(trimmed, []byte("{Gallery}")) || bytes.Equal(trimmed, []byte("{/Gallery}")) {
		reader.AdvanceLine()
		return parser.Close
	}

	gallery := node.(*GLightboxBlock)

	parts := bytes.SplitN(trimmed, []byte{'|'}, 2)
	url := bytes.TrimSpace(parts[0])

	caption := make([]byte, 0)
	if len(parts) > 1 {
		caption = bytes.TrimSpace(parts[1])
	}

	gallery.Images = append(gallery.Images, GLightboxImage{
		URL:     string(url),
		Caption: caption,
	})

	return parser.Continue | parser.NoChildren
}

func (p *GLightboxParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
}

func (p *GLightboxParser) CanInterruptParagraph() bool {
	return true
}

func (p *GLightboxParser) CanAcceptIndentedLine() bool {
	return false
}

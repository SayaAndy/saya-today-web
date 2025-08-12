package lightgallery

import (
	"bytes"
	"log/slog"
	"regexp"
	"strings"
	"time"

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

	if !bytes.HasPrefix(line, []byte("{Gallery")) {
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
		slog.Warn("invalid location specified for a gallery", slog.String("line", string(trimmed)), slog.String("location", string(parts[1])))
		return nil, parser.NoChildren
	}

	return &LightGalleryBlock{Location: loc}, parser.NoChildren
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

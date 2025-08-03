package tailwind

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type TailwindTransformer struct{}

func (t *TailwindTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Heading:
			classes := map[int]string{
				1: "font-patua text-[2.5vmax] font-bold text-main-dark mb-[1.2vmin] tracking-[0.04vmin]",
				2: "font-spectral text-[2vmax] font-bold text-main-dark mb-[0.8vmin] tracking-[0.04vmin]",
				3: "font-spectral text-[1.5vmax] font-medium text-main-dark mb-[0.6vmin] tracking-[0.04vmin]",
				4: "font-spectral text-[1.2vmax] font-medium text-main-medium mb-[0.4vmin] tracking-[0.04vmin]",
				5: "font-spectral text-[1vmax] font-medium text-main-medium mb-[0.4vmin] italic tracking-[0.04vmin]",
				6: "font-spectral text-[1vmax] font-medium text-secondary mb-[0.4vmin]",
			}
			if class, ok := classes[node.Level]; ok {
				node.SetAttribute([]byte("class"), []byte(class))
			}

		case *ast.Paragraph:
			node.SetAttribute([]byte("class"), []byte("text-[1vmax]/[2] font-spectral tracking-[0.04vmin] -indent-[2vmax] ml-[2vmax] mb-[1.6vmin]"))

		case *ast.List:
			if node.IsOrdered() {
				node.SetAttribute([]byte("class"), []byte("list-decimal list-inside space-y-[0.8vmin] mb-[1.6vmin] pl-[2vmax]"))
			} else {
				node.SetAttribute([]byte("class"), []byte("list-disc list-inside space-y-[0.8vmin] mb-[1.6vmin] pl-[2vmax]"))
			}

		case *ast.ListItem:
			node.SetAttribute([]byte("class"), []byte("text-[1vmax]/[2] font-spectral tracking-[0.04vmin]"))

		case *ast.Blockquote:
			node.SetAttribute([]byte("class"), []byte("border-l-[0.4vmin] border-main-medium bg-background-dark p-[0.8vmin] mb-[0.8vmin] italic"))

		case *ast.CodeSpan:
			node.SetAttribute([]byte("class"), []byte("bg-background-dark"))

		case *ast.CodeBlock, *ast.FencedCodeBlock:
			node.SetAttribute([]byte("class"), []byte("bg-background-dark p-[0.8vmin] rounded-lg overflow-x-auto mb-[1.6vmin]"))

		case *ast.Link:
			node.SetAttribute([]byte("class"), []byte("text-secondary hover:text-main-dark underline"))

		case *ast.Image:
			node.SetAttribute([]byte("class"), []byte("max-w-full h-auto rounded-lg shadow-lg mb-[1.6vmin]"))

		case *ast.Emphasis:
			switch node.Level {
			case 1:
				node.SetAttribute([]byte("class"), []byte("italic"))
			case 2:
				node.SetAttribute([]byte("class"), []byte("font-bold"))
			case 3:
				node.SetAttribute([]byte("class"), []byte("italic font-bold"))
			}
		}

		return ast.WalkContinue, nil
	})
}

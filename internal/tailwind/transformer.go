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
				1: "font-patua text-vmax2-5 font-bold text-main-dark mb-vmin1-2 ls-vmin0-04",
				2: "font-spectral text-vmax2 font-bold text-main-dark mb-vmin0-8 ls-vmin0-04",
				3: "font-spectral text-vmax1-5 font-medium text-main-dark mb-vmin0-6 ls-vmin0-04",
				4: "font-spectral text-vmax1-2 font-medium text-main-medium mb-vmin0-4 ls-vmin0-04",
				5: "font-spectral text-vmax1 font-medium text-main-medium mb-vmin0-4 italic ls-vmin0-04",
				6: "font-spectral text-vmax1 font-medium text-secondary mb-vmin0-4",
			}
			if class, ok := classes[node.Level]; ok {
				node.SetAttribute([]byte("class"), []byte(class))
			}

		case *ast.Paragraph:
			node.SetAttribute([]byte("class"), []byte("text-vmax1 font-spectral lh-2 ls-vmin0-04 timl-vmax2 mb-vmin1-6"))

		case *ast.List:
			if node.IsOrdered() {
				node.SetAttribute([]byte("class"), []byte("list-decimal list-inside space-y-vmin0-8 mb-vmin1-6 pl-vmax2"))
			} else {
				node.SetAttribute([]byte("class"), []byte("list-disc list-inside space-y-vmin0-8 mb-vmin1-6 pl-vmax2"))
			}

		case *ast.ListItem:
			node.SetAttribute([]byte("class"), []byte("text-vmax1 font-spectral lh-2 ls-vmin0-04"))

		case *ast.Blockquote:
			node.SetAttribute([]byte("class"), []byte("border-l-vmin0-4 border-main-medium bg-background-dark p-vmin0-8 mb-vmin0-8 italic"))

		case *ast.CodeSpan:
			node.SetAttribute([]byte("class"), []byte("bg-background-dark"))

		case *ast.CodeBlock, *ast.FencedCodeBlock:
			node.SetAttribute([]byte("class"), []byte("bg-background-dark p-vmin0-8 rounded-lg overflow-x-auto mb-vmin1-6"))

		case *ast.Link:
			node.SetAttribute([]byte("class"), []byte("text-secondary hover:text-main-dark underline"))

		case *ast.Image:
			node.SetAttribute([]byte("class"), []byte("max-w-full h-auto rounded-lg shadow-lg mb-vmin1-6"))

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

package tailwind

import (
	"bytes"
	"fmt"

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
				1: "font-andika text-4xl font-bold text-main-hard mb-3 tracking-[.0125rem]",
				2: "font-andika text-3xl font-bold text-main-hard mb-1 tracking-[.0125rem]",
				3: "font-andika text-2xl font-medium text-main-hard mb-0.8 tracking-[.0125rem]",
				4: "font-andika text-xl font-medium text-main-medium mb-0.5 tracking-[.0125rem]",
				5: "font-andika text-base font-medium text-main-medium mb-0.5 italic tracking-[.0125rem]",
				6: "font-andika text-base font-medium text-secondary mb-0.5",
			}
			if class, ok := classes[node.Level]; ok {
				node.SetAttribute([]byte("class"), []byte(class))
			}

		case *ast.Paragraph:
			node.SetAttribute([]byte("class"), []byte("text-base/[2] font-andika tracking-[.0125rem] -indent-8 ml-4 mb-8"))

		case *ast.List:
			if node.IsOrdered() {
				node.SetAttribute([]byte("class"), []byte("list-decimal list-inside space-y-2 mb-4 pl-8"))
			} else {
				node.SetAttribute([]byte("class"), []byte("list-disc list-inside space-y-2 mb-4 pl-8"))
			}

		case *ast.ListItem:
			node.SetAttribute([]byte("class"), []byte("text-base/[2] font-andika tracking-[.0125rem]"))

		case *ast.Blockquote:
			node.SetAttribute([]byte("class"), []byte("border-l-2 border-main-medium bg-background-dark p-1 mb-2 italic"))

		case *ast.CodeSpan:
			node.SetAttribute([]byte("class"), []byte("bg-background-dark"))

		case *ast.CodeBlock, *ast.FencedCodeBlock:
			node.SetAttribute([]byte("class"), []byte("bg-background-dark p-1 rounded-lg overflow-x-auto mb-4"))

		case *ast.Link:
			if bytes.HasPrefix(node.Destination, []byte{'.', '/'}) {
				onclickAttr := fmt.Appendf(make([]byte, 0, 21+len(node.Destination)), "return changeUrl('%s');", node.Destination)
				node.SetAttribute([]byte("onclick"), onclickAttr)
			}
			node.SetAttribute([]byte("class"), []byte("text-secondary hover:text-main-hard underline"))

		case *ast.Image:
			node.SetAttribute([]byte("class"), []byte("max-w-full h-auto rounded-lg shadow-lg mb-4"))

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

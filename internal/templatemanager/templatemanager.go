package templatemanager

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
)

type TemplateManager struct {
	templates map[string]templateManagerRender
}

type templateManagerRender struct {
	Main string
	Tmpl *template.Template
}

type TemplateManagerTemplates struct {
	Name  string
	Files []string
}

func NewTemplateManager(templates []TemplateManagerTemplates) (*TemplateManager, error) {
	templateMap := make(map[string]templateManagerRender)

	for _, tmplStruct := range templates {
		tmpl := template.New("").Funcs(template.FuncMap{
			"contains": strings.Contains,
		})
		tmpl, err := tmpl.ParseFiles(tmplStruct.Files...)
		if err != nil {
			return nil, err
		}
		templateMap[tmplStruct.Name] = templateManagerRender{
			Main: filepath.Base(tmplStruct.Files[0]),
			Tmpl: tmpl,
		}

	}

	return &TemplateManager{
		templates: templateMap,
	}, nil
}

func (tm *TemplateManager) Render(name string, data interface{}) ([]byte, error) {
	tmpl, exists := tm.templates[name]
	if !exists {
		return nil, fmt.Errorf("template %s is not found", name)
	}

	var buf bytes.Buffer
	err := tmpl.Tmpl.ExecuteTemplate(&buf, tmpl.Main, data)
	return buf.Bytes(), err
}

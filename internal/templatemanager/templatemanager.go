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

var templateFuncMap = template.FuncMap{
	"contains": strings.Contains,
	"iterate": func(count uint) []uint {
		items := make([]uint, count)
		for i := range count {
			items[i] = i
		}
		return items
	},
	"replace": strings.ReplaceAll,
}

func NewTemplateManager(templates ...TemplateManagerTemplates) (*TemplateManager, error) {
	templateMap := make(map[string]templateManagerRender)

	for _, tmplStruct := range templates {
		tmpl := template.New(tmplStruct.Name).Funcs(templateFuncMap)
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

func (tm *TemplateManager) Render(name string, data any, files ...string) ([]byte, error) {
	tmpl, exists := tm.templates[name]
	if !exists {
		return nil, fmt.Errorf("template %s is not found", name)
	}

	var err error
	var tempTmpl *template.Template
	if len(files) == 0 {
		tempTmpl = tmpl.Tmpl
	} else {
		tempTmpl, err = tmpl.Tmpl.Clone()
		if err != nil {
			return nil, fmt.Errorf("couldn't clone existing template for rendering: %w", err)
		}
		tempTmpl, err = tempTmpl.ParseFiles(files...)
		if err != nil {
			return nil, fmt.Errorf("couldn't include additional files in template rendering: %w", err)
		}
	}

	var buf bytes.Buffer
	err = tempTmpl.ExecuteTemplate(&buf, tmpl.Main, data)
	return buf.Bytes(), err
}

func (tm *TemplateManager) Add(name string, files ...string) error {
	if len(files) == 0 {
		return fmt.Errorf("you can't add template without any files")
	}

	tmpl := template.New(name).Funcs(templateFuncMap)

	tmpl, err := tmpl.ParseFiles(files...)
	if err != nil {
		return fmt.Errorf("failed to add template into manager: %w", err)
	}

	tm.templates[name] = templateManagerRender{
		Main: filepath.Base(files[0]),
		Tmpl: tmpl,
	}
	return nil
}

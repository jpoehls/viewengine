package viewengine

import (
	"bytes"
	"html/template"
)

var builtins = template.FuncMap{
	// optional_template will be assigned before the template set is rendered.
	"optional_template": placeholder,
	"htmlEncode":        template.HTMLEscaper,
}

func placeholder(...interface{}) string {
	return ""
}

func optionalTemplate(t *template.Template) func(string, ...interface{}) (template.HTML, error) {
	return func(name string, data ...interface{}) (template.HTML, error) {
		var tm = t.Lookup(name)
		if tm != nil {
			var b bytes.Buffer
			err := tm.ExecuteTemplate(&b, name, data)
			if err != nil {
				return template.HTML(""), err
			}
			return template.HTML(b.String()), nil
		} else {
			return template.HTML(""), nil
		}
	}
}

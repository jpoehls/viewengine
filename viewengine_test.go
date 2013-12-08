package viewengine

import (
	"bytes"
	"strings"
	"testing"
)

func showWhitespace(input string) string {
	return strings.Replace(strings.Replace(input, " ", "·", -1), "\t", "⇥", -1)
}

// Renders a template to a string.
func (ve *ViewEngine) render(name string, data interface{}) (string, error) {
	var out bytes.Buffer
	err := ve.Execute(&out, name, data)
	if err != nil {
		return "", err
	}

	return out.String(), nil
}

func Test_SimpleMasterPage(t *testing.T) {
	ve := New()

	Must(ve.Parse("masterpage", `<html>
	<body>
		{{template "__body"}}
	</body>
</html>`))

	Must(ve.Parse("homepage", `{{define "__body"}}<p>Page content here.</p>{{end}}
{{template "masterpage"}}`))

	out, err := ve.render("homepage", nil)
	if err != nil {
		t.Fatal(err)
	}

	if out != `
<html>
	<body>
		<p>Page content here.</p>
	</body>
</html>` {
		t.Logf("\"%s\"", out)
		t.Fatal("Rendered template is incorrect.")
	}
}

func Test_SimpleMasterPageWithIncludes(t *testing.T) {
	ve := New()

	Must(ve.Parse("masterpage", `<html>
	<body>
		{{template "__body"}}
		{{template "snippet"}}
	</body>
</html>`))

	Must(ve.Parse("homepage", `{{define "__body"}}<p>Page content here. {{template "snippet"}}</p>{{end}}
{{template "masterpage"}}`))

	Must(ve.Parse("snippet", `SNIPPET`))

	out, err := ve.render("homepage", nil)
	if err != nil {
		t.Fatal(err)
	}

	if out != `
<html>
	<body>
		<p>Page content here. SNIPPET</p>
		SNIPPET
	</body>
</html>` {
		t.Logf("\"%s\"", out)
		t.Fatal("Rendered template is incorrect.")
	}
}

func Test_NestedMasterPage(t *testing.T) {
	ve := New()

	Must(ve.Parse("masterpage_top", `<html>
	<body>
		{{template "__body"}}
	</body>
</html>`))

	Must(ve.Parse("masterpage_inner", `{{define "__body"}}<h1>Common Header</h1>
	{{template "__body"}}{{end}}
{{template "masterpage_top"}}`))

	Must(ve.Parse("homepage", `{{define "__body"}}<p>Page content here.</p>{{end}}
{{template "masterpage_inner"}}`))

	out, err := ve.render("homepage", nil)
	if err != nil {
		t.Fatal(err)
	}

	if out != `
<html>
	<body>
		<h1>Common Header</h1>
		<p>Page content here.</p>
	</body>
</html>` {
		t.Logf("\"%s\"", out)
		t.Fatal("Rendered template is incorrect.")
	}
}

func Test_MasterPageWithOptionalSections(t *testing.T) {
	ve := New()

	Must(ve.Parse("masterpage", `<html>
	<body>
		{{optional_template "__header"}}
		{{template "__body"}}
		{{optional_template "__footer"}}
	</body>
</html>`))

	Must(ve.Parse("homepage", `{{define "__body"}}<p>Page content here.</p>{{end}}
{{define "__header"}}<h1>Page header</h1>{{end}}
{{template "masterpage"}}`))

	out, err := ve.render("homepage", nil)
	if err != nil {
		t.Fatal(err)
	}

	if out != `

<html>
	<body>
		<h1>Page header</h1>
		<p>Page content here.</p>

	</body>
</html>` {
		t.Logf("\"%s\"", out)
		t.Fatal("Rendered template is incorrect.")
	}
}

func Test_DataBinding(t *testing.T) {
	ve := New()

	Must(ve.Parse("masterpage", `<html>
	<body>
		{{template "__body" .}}
		{{.}}
	</body>
</html>`))

	Must(ve.Parse("homepage", `{{define "__body"}}<p>Page content here. {{.}}</p>{{end}}
{{.}}
{{template "masterpage" .}}
{{.}}`))

	data := "DATA"

	out, err := ve.render("homepage", data)
	if err != nil {
		t.Fatal(err)
	}

	if out != `
DATA
<html>
	<body>
		<p>Page content here. DATA</p>
		DATA
	</body>
</html>
DATA` {
		t.Logf("\"%s\"", showWhitespace(out))
		t.Fatal("Rendered template is incorrect.")
	}
}

func Test_ParseFiles(t *testing.T) {
	ve := New()

	Must(ve.ParseGlob("./test_views/", "**/*.gohtml"))
	//Must(ve.ParseGlob("./test_views/", "*.gohtml"))
	//Must(ve.ParseFiles("./test_views/", "confirmation_email.gohtml"))
}

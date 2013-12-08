package viewengine

import (
	"bytes"
	"testing"
)

func Test_SimpleParseAndExecute(t *testing.T) {

	ve := New()

	Must(ve.Parse("main_layout", `The subject is: {{template "__subject"}}
The message is: {{template "__content"}}`))

	Must(ve.Parse("menu", `MENU`))

	Must(ve.Parse("page1", `{{define "__subject"}}Page 1 Subject{{end}}
{{define "__content"}}Page 1 Content{{end}}
{{optional_template "menu"}}
{{template "main_layout"}}`))

	var out bytes.Buffer
	err := ve.Execute(&out, "page1", nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(out.String())
}

func Test_ParseFiles(t *testing.T) {
	ve := New()

	Must(ve.ParseGlob("./test_views/", "**/*.gohtml"))
	//Must(ve.ParseGlob("./test_views/", "*.gohtml"))
	//Must(ve.ParseFiles("./test_views/", "confirmation_email.gohtml"))
}

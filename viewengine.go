/*
TODO
- Add tests
- Add docs
- Consider a different naming for "partials" vs "pages". Those aren't very descriptive.
- Add optional polling for template file changes or file watching.
- Lots of docs and examples.
*/

/*
.gohtml, .gomaster
.gohtml can {{template ""}} include other .gohtml files and .gomaster files
.gomaster can {{template ""}} include other .gohtml files

Parse - add view with specified name and code
ParseFiles - add all views from given file names
ParseGlob - add all views matching given glob

- Get parse tree of template.
- If any templates are {{define}}'d then change their
  name to be prefixed with the top-level template name.
  (So that pages can {{define "head"}}, "body", etc.)
- Get names of all templates it references.
- Store the parse tree along with the list of template names referenced.
- Execute should add the named template and all of its dependencies
  to a template and then render it.
- Cache templates created during execution.
*/

/*
Content sections in master pages must be named with a __ prefix.
*What happens if they aren't?*

EXAMPLES
- CANNOT use nested master pages.
- CANNOT include a page that uses a master page inside another page.
- Master page
	master (include a partial)
	content page (using master, include another partial)
- Including a partial
	content page (no master, include a partial)
- Rendering a partial or simple view
	partial (no master, no other partials)
- Master pages with optional content placeholders
	master (defines two optional_template sections)
	content page (using master, only fills one of the optionals)
- Passing data to a view
	master (uses some data)
	content page (using master, uses some data)

- Parsing all view files in a folder, recursive (glob)
- Parsing specific view files
- Parsing views from strings
*/

package viewengine

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"text/template/parse"
)

// Suffix used by master page file names.
// The Parse* functions will use this suffix
// to determine whether the file is a regular view
// or a master page view.
var MasterPageSuffix = ".master.gohtml"

type ViewEngine struct {
	// A partial is any view that DOES NOT
	// define any content sections.
	partials *template.Template

	// A page is any view that DOES
	// define a content section.
	// Example: {{define "__sectionName"}} ... {{end}}
	//
	// We can't include pages in the partials template set
	// because they will likely define duplicate content
	// section templates such as __header, etc.
	pages map[string]map[string]*parse.Tree

	mu sync.Mutex
}

// New allocates a new view engine.
func New() *ViewEngine {
	ve := &ViewEngine{}
	ve.partials = template.Must(template.New("").Parse(""))
	ve.partials.Funcs(builtins)
	ve.pages = make(map[string]map[string]*parse.Tree)
	return ve
}

// Must is a helper that wraps a call to a function returning (*ViewEngine, error)
// and panics if the error is non-nil. It is intended for use in variable initializations
// such as:
//	var ve = viewengine.Must(viewengine.New().Parse("index", "..."))
func Must(ve *ViewEngine, err error) *ViewEngine {
	if err != nil {
		panic(err)
	}
	return ve
}

func normalizeName(name string) string {
	return strings.TrimPrefix(name, "/")
}

// Parse parses a string into a template.
func (ve *ViewEngine) Parse(name, src string) (*ViewEngine, error) {
	name = normalizeName(name)

	ve.mu.Lock()
	defer ve.mu.Unlock()

	// Error if a template by this name has already been added.
	_, ok := ve.pages[name]
	if ok || ve.partials.Lookup(name) != nil {
		return ve, fmt.Errorf("viewengine: redefinition of template %q", name)
	}

	trees, err := parse.Parse(name, src, "", "", builtins)
	if err != nil {
		return ve, err
	}

	isPage := false

	// If any of the templates start with the contentSectionPrefix
	// then the template set is treated as a page.
	for _, v := range trees {
		if strings.HasPrefix(v.Name, contentSectionPrefix) {
			isPage = true
			break
		}
	}

	if isPage {
		log.Printf("viewengine: adding page %q\n", name)
		ve.pages[name] = trees
	} else {
		for _, v := range trees {
			log.Printf("viewengine: adding partial %q\n", v.Name)
			_, err := ve.partials.AddParseTree(v.Name, v)
			if err != nil {
				return ve, err
			}
		}
	}

	return ve, nil
}

// ParseFiles parses the template definitions from the named files.
// There must be at least one file.
// *.gohtml
func (ve *ViewEngine) ParseFiles(root string, filenames ...string) (*ViewEngine, error) {
	if len(filenames) == 0 {
		// Not really a problem, but be consistent.
		return ve, fmt.Errorf("viewengine: no files named in call to ParseFiles")
	}

	for _, filename := range filenames {
		srcBytes, err := ioutil.ReadFile(filepath.Join(root, filename))
		if err != nil {
			return ve, err
		}
		src := string(srcBytes)

		_, err = ve.Parse(filename, src)
		if err != nil {
			return ve, err
		}
	}

	return ve, nil
}

// ParseGlob parses the template definitions in the files identified by the
// pattern and associates the resulting templates with ve. The pattern is
// processed by filepath.Glob and must match at least one file. ParseGlob is
// equivalent to calling ve.ParseFiles with the list of files matched by the
// pattern.
func (ve *ViewEngine) ParseGlob(root, pattern string) (*ViewEngine, error) {
	filenames, err := filepath.Glob(filepath.Join(root, pattern))
	if err != nil {
		return ve, err
	}
	if len(filenames) == 0 {
		return ve, fmt.Errorf("viewengine: pattern matches no files: %#q", pattern)
	}
	for i := range filenames {
		filenames[i] = strings.TrimPrefix(filenames[i], filepath.Clean(root))
	}
	return ve.ParseFiles(root, filenames...)
}

// Execute applies the template associated with ve that has the given
// name to the specified data object and writes the output to wr.
func (ve *ViewEngine) Execute(wr io.Writer, name string, data interface{}) error {
	name = normalizeName(name)

	ve.mu.Lock()
	page, ok := ve.pages[name]
	ve.mu.Unlock()
	if ok {
		// Clone our set of partials into a new namespace.
		renderSet, err := ve.partials.Clone()
		if err != nil {
			return err
		}

		renderSet.Funcs(template.FuncMap{
			"optional_template": optionalTemplate(renderSet),
		})

		// Add the page's templates into the new namespace.
		for _, tree := range page {
			// Rename the top-level template to ~page.
			tname := tree.Name
			if tname == name {
				tname = "~page"
			}

			_, err := renderSet.AddParseTree(tname, tree)
			if err != nil {
				return err
			}
		}

		return renderSet.ExecuteTemplate(wr, "~page", data)
	} else {
		return ve.partials.ExecuteTemplate(wr, name, data)
	}
}

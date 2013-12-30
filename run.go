package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"
	"text/template/parse"
)

func main() {

	// TODO: MountDir() method takes a directory path
	//       and does a ParseGlob() on it.
	//       Templates loaded from the dir have their names
	//       shortened to just the relative path.
	//       Ex. MountDir("/path/to/dir") might add a template "blah.gohtml"
	//       rather than "/path/to/dir/blah.gohtml".

	// TODO: NoCache = true; should cause ParseGlob, ParseFiles, and MountDir
	//       to always reread the template from disk when executing it.
	//       Basically, the calls to those parse methods cause
	//       the file names to be cached and if NoCache == true then
	//       Execute() reloads the template set from scratch.

	/*
		Goal is to load all view files into a single
		template set and transparently handle any
		master page content area naming collisions across pages.
	*/

	var err error
	var masterPageTemplates []string
	var tmpl = template.New("~")

	// Parse an empty template to avoid panics due
	// to an uninitialized internal parse tree.
	// https://groups.google.com/forum/#!topic/golang-nuts/C8jPINo8sHc
	_, _ = tmpl.Parse("")

	filenames, err := filepath.Glob("./test_views/views/simple/*.gohtml")
	if err != nil {
		panic(err)
	}

	for _, filename := range filenames {
		fmt.Printf("file: %s\n", filename)

		// Read file.
		srcBytes, err := ioutil.ReadFile(filename)
		if err != nil {
			panic(err)
		}
		src := string(srcBytes)

		tree, err := parse.Parse(filename, src, "", "", nil)
		if err != nil {
			panic(err)
		}

		// Iterate all templates defined in the file.
		for _, fileTemplate := range tree {

			// Preprocess page templates.
			// Do nothing to master page templates.
			if !strings.HasSuffix(filename, ".master.gohtml") {
				processPageTemplate(filename, fileTemplate)
			} else {
				// Keep track of which templates were from
				// master page template files.
				masterPageTemplates = append(masterPageTemplates, fileTemplate.Name)
			}

			// Add to our global template set.
			_, err = tmpl.AddParseTree(fileTemplate.Name, fileTemplate)
			if err != nil {
				panic(err)
			}
		}
		fmt.Println()
	}

	// Execute a template as a test.
	var templateName = "test_views/views/simple/page1.gohtml"
	// Clone the template set.
	execTmpl, err := tmpl.Clone()
	if err != nil {
		panic(err)
	}

	// Update the ~ {{template}} calls in all
	// master page templates to have the file name prefix.
	for _, masterPageTemplateName := range masterPageTemplates {
		var execTmplItem = execTmpl.Lookup(masterPageTemplateName)
		prefixTildeTemplates(templateName, execTmplItem.Root.Nodes)
	}

	// *Then* execute the template.
	// We can cache the template set at this point for future renders of this view.
	fmt.Println("*************************")
	var renderedOutput = &bytes.Buffer{}
	err = execTmpl.ExecuteTemplate(renderedOutput, templateName, nil)
	if err != nil {
		panic(err)
	}
	fmt.Print(renderedOutput.String())
	fmt.Println()
	fmt.Println("*************************")
}

func processPageTemplate(filename string, tree *parse.Tree) {
	// Prefix ~ template names with the file name.
	// ~ denotes a template used as a master page content section.
	// We prefix these with the file name to prevent 'redefinition of template'
	// errors when we store all of the templates from multiple files
	// in the same template set.
	if strings.HasPrefix(tree.Name, "~") {
		tree.Name = filename + tree.Name
	}
	if strings.HasPrefix(tree.ParseName, "~") {
		tree.ParseName = filename + tree.ParseName
	}
	fmt.Printf("\tdefine: %s\n", tree.Name)

	// Iterate the parse tree and update any
	// {{template "~..."}} nodes to use the file name prefix.
	prefixTildeTemplates(filename, tree.Root.Nodes)
}

func prefixTildeTemplates(filename string, nodes []parse.Node) {
	for _, node := range nodes {

		switch n := node.(type) {
		case *parse.TemplateNode:
			if strings.HasPrefix(n.Name, "~") {
				n.Name = filename + n.Name
			}

		case *parse.IfNode:
			prefixTildeTemplates(filename, n.List.Nodes)
			if n.ElseList != nil {
				prefixTildeTemplates(filename, n.ElseList.Nodes)
			}

		case *parse.RangeNode:
			prefixTildeTemplates(filename, n.List.Nodes)
			if n.ElseList != nil {
				prefixTildeTemplates(filename, n.ElseList.Nodes)
			}

		case *parse.WithNode:
			prefixTildeTemplates(filename, n.List.Nodes)
			if n.ElseList != nil {
				prefixTildeTemplates(filename, n.ElseList.Nodes)
			}

		case *parse.ListNode:
			prefixTildeTemplates(filename, n.Nodes)
		}
	}
}

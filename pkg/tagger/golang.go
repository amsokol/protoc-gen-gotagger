package tagger

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"unicode"

	"github.com/fatih/structtag"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
)

type goStruct map[string]*structtag.Tags

type goFile struct {
	structs map[string]goStruct
}

func (p *plugin) toGolangStructName(parents []string, name string) string {
	names := make([]string, len(parents), len(parents)+1)
	copy(names, parents)
	names = append(names, name)

	var n string
	for _, v := range names {
		var uppercased bool
		r := []rune(v)
		if unicode.IsLower(r[0]) {
			uppercased = true
			r[0] = unicode.ToUpper(r[0])
		}
		v = string(r)
		if len(n) > 0 && !uppercased {
			n += "_"
		}

		n += v
	}

	return n
}

func (p *plugin) modifyTargetFiles() error {
	fset := token.NewFileSet()

	for name, file := range p.targetFiles {
		path := p.outputPath + "/" + name

		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed parse Go file '%s': %s", path, err.Error())
		}

		if err = updateTags(f, file.structs); err != nil {
			return fmt.Errorf("failed to update tags in Go file '%s': %s", path, err.Error())
		}

		var buf bytes.Buffer
		if err = format.Node(&buf, fset, f); err != nil {
			return fmt.Errorf("failed to store updated Go file '%s': %s", path, err.Error())
		}

		content := buf.String()
		p.response.File = append(p.response.File, &plugin_go.CodeGeneratorResponse_File{
			Name:    &name,
			Content: &content,
		})
	}

	return nil
}

// The following code has been got from here:
// https://github.com/srikrsna/protoc-gen-gotag/blob/master/module/replace.go

// updateTags updates the existing tags with the map passed and modifies existing tags if any of the keys are matched.
// First key to the tags argument is the name of the struct, the second key corresponds to field names.
func updateTags(n ast.Node, tags map[string]goStruct) error {
	r := retag{}
	f := func(n ast.Node) ast.Visitor {
		if r.err != nil {
			return nil
		}

		if tp, ok := n.(*ast.TypeSpec); ok {
			r.tags = tags[tp.Name.String()]
			return r
		}

		return nil
	}

	ast.Walk(structVisitor{f}, n)

	return r.err
}

type structVisitor struct {
	visitor func(n ast.Node) ast.Visitor
}

func (v structVisitor) Visit(n ast.Node) ast.Visitor {
	if tp, ok := n.(*ast.TypeSpec); ok {
		if _, ok := tp.Type.(*ast.StructType); ok {
			ast.Walk(v.visitor(n), n)
			return nil // This will ensure this struct is no longer traversed
		}
	}
	return v
}

type retag struct {
	err  error
	tags map[string]*structtag.Tags
}

func (v retag) Visit(n ast.Node) ast.Visitor {
	if v.err != nil {
		return nil
	}

	if f, ok := n.(*ast.Field); ok {
		if len(f.Names) == 0 {
			return nil
		}
		newTags := v.tags[f.Names[0].String()]
		if newTags == nil {
			return nil
		}

		if f.Tag == nil {
			f.Tag = &ast.BasicLit{
				Kind: token.STRING,
			}
		}

		oldTags, err := structtag.Parse(strings.Trim(f.Tag.Value, "`"))
		if err != nil {
			v.err = err
			return nil
		}

		for _, t := range newTags.Tags() {
			oldTags.Set(t)
		}

		f.Tag.Value = "`" + oldTags.String() + "`"

		return nil
	}

	return v
}

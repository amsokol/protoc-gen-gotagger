package tagger

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
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
		if err = printer.Fprint(&buf, fset, f); err != nil {
			return fmt.Errorf("failed to store updated Go file '%s': %s", path, err.Error())
		}

		content := string(buf.Bytes())
		p.response.File = append(p.response.File, &plugin_go.CodeGeneratorResponse_File{
			Name:    &name,
			Content: &content,
		})
	}

	return nil
}

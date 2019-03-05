package tagger

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/fatih/structtag"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

func (p *plugin) analyzeSourceFiles() error {
	for _, f := range p.request.ProtoFile {
		var generate bool
		for _, g := range p.request.FileToGenerate {
			if g == *f.Name {
				generate = true
				break
			}
		}

		if generate {
			if err := p.analyzeFile(f); err != nil {
				return fmt.Errorf("failed to analyze proto file '%s': %s", *f.Name, err.Error())
			}
		}
	}

	return nil
}

func (p *plugin) analyzeFile(f *descriptor.FileDescriptorProto) error {
	if f.Syntax != nil && *f.Syntax != "proto3" {
		return fmt.Errorf("unsupported syntax '%s', must be 'proto3'", *f.Syntax)
	}

	file := goFile{structs: map[string]goStruct{}}

	for _, m := range f.MessageType {
		if err := p.analyzeMessageType(file, []string{}, m); err != nil {
			return fmt.Errorf("failed to analyze message type '%s': %s", *m.Name, err.Error())
		}
	}

	if len(file.structs) > 0 {
		n := filepath.Base(*f.Name)
		n = strings.TrimSuffix(n, filepath.Ext(n))
		p.targetFiles[n+".pb.go"] = file
	}

	return nil
}

func (p *plugin) analyzeMessageType(file goFile, parents []string, message *descriptor.DescriptorProto) error {
	s := goStruct{}
	goMes := p.toGolangStructName(parents, *message.Name)

	if p.xxxTag != nil {
		s["XXX_NoUnkeyedLiteral"] = p.xxxTag
		s["XXX_unrecognized"] = p.xxxTag
		s["XXX_sizecache"] = p.xxxTag
	}

	for _, field := range message.Field {
		ext, err := p.getExtension(field.GetOptions(), E_Tags)
		if err != nil {
			return fmt.Errorf("failed to get extension for field '%s' type '%s': %s",
				*field.Name, p.getMessageURI(parents, *message.Name), err.Error())
		}
		if len(ext) > 0 {
			tags, err := structtag.Parse(ext)
			if err != nil {
				return fmt.Errorf("failed to parse XXX tags '%s': %s", ext, err.Error())
			}

			n := p.toGolangFieldName(*field.Name)
			if field.OneofIndex != nil {
				oneOf := goStruct{}
				oneOf[n] = tags
				file.structs[goMes+"_"+n] = oneOf
			} else {
				s[n] = tags
			}
		}
	}

	for _, oneOf := range message.GetOneofDecl() {
		ext, err := p.getExtension(oneOf.GetOptions(), E_OneofTags)
		if err != nil {
			return fmt.Errorf("failed to get extension for oneof '%s' type '%s': %s",
				*oneOf.Name, p.getMessageURI(parents, *message.Name), err.Error())
		}
		if len(ext) > 0 {
			tags, err := structtag.Parse(ext)
			if err != nil {
				return fmt.Errorf("failed to parse XXX tags '%s': %s", ext, err.Error())
			}
			s[p.toGolangFieldName(*oneOf.Name)] = tags
		}
	}

	for _, m := range message.NestedType {
		ps := p.addMessageParent(parents, *message.Name)
		if err := p.analyzeMessageType(file, ps, m); err != nil {
			return fmt.Errorf("failed to analyze message type '%s': %s", p.getMessageURI(ps, *m.Name), err.Error())
		}
	}

	if len(s) > 0 {
		file.structs[goMes] = s
	}

	return nil
}

// Following code has been copied from here:
// https://github.com/lyft/protoc-gen-star/blob/master/extension.go
func (p *plugin) getExtension(opts proto.Message, ext *proto.ExtensionDesc) (string, error) {
	if opts == nil {
		return "", nil
	}

	if !proto.HasExtension(opts, ext) {
		return "", nil
	}

	val, err := proto.GetExtension(opts, ext)
	if err != nil {
		return "", fmt.Errorf("failed to get extension: %s", err.Error())
	}

	v := reflect.ValueOf(val)
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	val = v.Interface()

	s, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("cannot assign extension type '%q' to output type 'string'", v.Type().String())
	}

	return s, nil
}

func (p *plugin) getMessageURI(parents []string, message string) string {
	var res string
	for _, s := range parents {
		res += s + "."
	}
	return res + message
}

func (p *plugin) addMessageParent(parents []string, parent string) []string {
	res := make([]string, 0, len(parents)+1)
	for _, s := range parents {
		res = append(res, s)
	}
	return append(res, parent)
}

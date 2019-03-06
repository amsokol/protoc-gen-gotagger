package tagger

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/fatih/structtag"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/amsokol/protoc-gen-gotagger/proto/tagger"
)

// analyzeSourceFiles scans source proto files one by one (calls plugin.analyzeFile func) to extract field tags
func (p *plugin) analyzeSourceFiles() error {
	for _, f := range p.request.GetProtoFile() {
		var generate bool
		for _, g := range p.request.GetFileToGenerate() {
			if g == f.GetName() {
				generate = true
				break
			}
		}

		if generate {
			if err := p.analyzeFile(f); err != nil {
				return fmt.Errorf("failed to analyze proto file '%s': %s", f.GetName(), err.Error())
			}
		}
	}

	return nil
}

// analyzeFile scans source proto file (provided by 'f') to extract field tags
// It proccess each proto message in the file one by one to find field tags.
// In case on found it stores tags in plugin.targetFiles map to update Go files on the next phases.
func (p *plugin) analyzeFile(f *descriptor.FileDescriptorProto) error {
	if f.GetSyntax() != "proto3" {
		return fmt.Errorf("unsupported syntax '%s', must be 'proto3'", f.GetSyntax())
	}

	file := goFile{structs: map[string]goStruct{}}

	for _, m := range f.GetMessageType() {
		if err := p.analyzeMessageType(file, []string{}, m); err != nil {
			return fmt.Errorf("failed to analyze message type '%s': %s", m.GetName(), err.Error())
		}
	}

	if len(file.structs) > 0 {
		n := filepath.Base(f.GetName())
		n = strings.TrimSuffix(n, filepath.Ext(n))
		p.targetFiles[n+".pb.go"] = file
	}

	return nil
}

// analyzeMessageType analyze proto Message:
// - extracting field tags
// - extracting OneOf tags
// It drills down into nested proto Messages also.
func (p *plugin) analyzeMessageType(file goFile, parents []string, message *descriptor.DescriptorProto) error {
	s := goStruct{}
	goMes := p.toGolangStructName(parents, message.GetName())

	if p.xxxTags != nil {
		s["XXX_NoUnkeyedLiteral"] = p.xxxTags
		s["XXX_unrecognized"] = p.xxxTags
		s["XXX_sizecache"] = p.xxxTags
	}

	// scan proto message fields
	for _, field := range message.GetField() {
		ext, err := p.getExtension(field.GetOptions(), tagger.E_Tags)
		if err != nil {
			return fmt.Errorf("failed to get extension for field '%s' type '%s': %s",
				field.GetName(), p.getMessageURI(parents, message.GetName()), err.Error())
		}
		if len(ext) > 0 {
			tags, err := structtag.Parse(ext)
			if err != nil {
				return fmt.Errorf("failed to parse XXX tags '%s': %s", ext, err.Error())
			}

			n := p.toGolangFieldName(field.GetName())
			if field.OneofIndex != nil {
				oneOf := goStruct{}
				oneOf[n] = tags
				file.structs[goMes+"_"+n] = oneOf
			} else {
				s[n] = tags
			}
		}
	}

	// scan proto message oneOfs
	for _, oneOf := range message.GetOneofDecl() {
		ext, err := p.getExtension(oneOf.GetOptions(), tagger.E_OneofTags)
		if err != nil {
			return fmt.Errorf("failed to get extension for oneof '%s' type '%s': %s",
				oneOf.GetName(), p.getMessageURI(parents, message.GetName()), err.Error())
		}
		if len(ext) > 0 {
			tags, err := structtag.Parse(ext)
			if err != nil {
				return fmt.Errorf("failed to parse XXX tags '%s': %s", ext, err.Error())
			}
			s[p.toGolangFieldName(oneOf.GetName())] = tags
		}
	}

	// scan nested proto messages
	for _, m := range message.GetNestedType() {
		ps := make([]string, len(parents), len(parents)+1)
		copy(ps, parents)
		ps = append(ps, message.GetName())
		if err := p.analyzeMessageType(file, ps, m); err != nil {
			return fmt.Errorf("failed to analyze message type '%s': %s", p.getMessageURI(ps, m.GetName()), err.Error())
		}
	}

	if len(s) > 0 {
		file.structs[goMes] = s
	}

	return nil
}

// getExtension extract tags (proto extension) from field options.
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

// getMessageURI construct message URI is used for error logging
// Example of proto:
// message Data1 {
//	messafe Data2 {
//	 ... fields
//	}
// }
// So URI of Data2 proto mesage is 'Data1.Data2'.
func (p *plugin) getMessageURI(parents []string, message string) string {
	var res string
	for _, s := range parents {
		res += s + "."
	}
	return res + message
}

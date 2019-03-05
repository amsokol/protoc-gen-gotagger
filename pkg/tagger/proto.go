package tagger

import (
	"fmt"
	"path/filepath"
	"strings"

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

	if p.xxxTag != nil {
		s["XXX_NoUnkeyedLiteral"] = p.xxxTag
		s["XXX_unrecognized"] = p.xxxTag
		s["XXX_sizecache"] = p.xxxTag
	}

	for _, m := range message.NestedType {
		ps := p.addMessageParent(parents, *message.Name)
		if err := p.analyzeMessageType(file, ps, m); err != nil {
			return fmt.Errorf("failed to analyze message type '%s': %s", p.getMessageURI(ps, *m.Name), err.Error())
		}
	}

	if len(s) > 0 {
		file.structs[p.toGolangStructName(parents, *message.Name)] = s
	}

	return nil
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

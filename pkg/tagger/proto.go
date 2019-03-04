package tagger

import (
	"fmt"
	"log"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

func (p *plugin) analyzeSources() error {
	for _, f := range p.request.ProtoFile {
		if err := p.analyzeFile(f); err != nil {
			return fmt.Errorf("failed to analyze proto file '%s': %s", *f.Name, err.Error())
		}
	}

	return nil
}

func (p *plugin) analyzeFile(f *descriptor.FileDescriptorProto) error {
	// TODO: DEBUG
	log.Printf("source file: %s", *f.Name)

	if f.Syntax != nil && *f.Syntax != "proto3" {
		return fmt.Errorf("unsupported syntax '%s', must be 'proto3'", *f.Syntax)
	}

	for _, m := range f.MessageType {
		if err := p.analyzeMessageType([]string{}, m); err != nil {
			return fmt.Errorf("failed to analyze message type '%s': %s", *m.Name, err.Error())
		}
	}

	return nil
}

func (p *plugin) analyzeMessageType(parents []string, message *descriptor.DescriptorProto) error {
	// TODO: DEBUG
	log.Printf("message type: %s", p.getMessageURI(parents, *message.Name))

	fields := goStructFields{}

	if len(p.xxxTag) > 0 {
		fields["XXX_NoUnkeyedLiteral"] = p.xxxTag
		fields["XXX_unrecognized"] = p.xxxTag
		fields["XXX_sizecache"] = p.xxxTag
	}

	for _, m := range message.NestedType {
		ps := p.addMessageParent(parents, *message.Name)
		if err := p.analyzeMessageType(ps, m); err != nil {
			return fmt.Errorf("failed to analyze message type '%s': %s", p.getMessageURI(ps, *m.Name), err.Error())
		}
	}

	if len(fields) > 0 {
		p.goStructs[p.toGolangStructName(parents, *message.Name)] = fields
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

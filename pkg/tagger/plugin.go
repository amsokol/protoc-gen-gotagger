package tagger

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
)

// Plugin is interface to proccess files
type Plugin interface {
	// Proccess input data and store result to the output writer
	Proccess() error
}

// NewPlugin returns object is implementing Plugin interface
func NewPlugin(in io.Reader, out io.Writer) Plugin {
	return &plugin{
		in:       in,
		request:  &plugin_go.CodeGeneratorRequest{},
		response: &plugin_go.CodeGeneratorResponse{},
		out:      out,
	}
}

type plugin struct {
	in      io.Reader
	request *plugin_go.CodeGeneratorRequest

	response *plugin_go.CodeGeneratorResponse
	out      io.Writer
}

func (p *plugin) error(msg string, args ...interface{}) error {
	s := fmt.Sprintf(msg, args...)
	p.response.Error = &s
	return p.write()
}

func (p *plugin) read() error {
	data, err := ioutil.ReadAll(p.in)
	if err != nil {
		return fmt.Errorf("failed to read marshaled request from input: %s", err.Error())
	}

	if err = proto.Unmarshal(data, p.request); err != nil {
		return fmt.Errorf("failed to unmarshal request from binary data: %s", err.Error())
	}

	// TODO: remove DEBUG
	// ioutil.WriteFile("./stdin.bin", data, 0)

	return nil
}

func (p *plugin) write() error {
	data, err := proto.Marshal(p.response)
	if err != nil {
		return fmt.Errorf("failed marshal response to binary data: %s", err.Error())
	}

	_, err = p.out.Write(data)
	if err != nil {
		return fmt.Errorf("failed write marshaled response to output: %s", err.Error())
	}

	return nil
}

func (p *plugin) Proccess() error {
	if err := p.read(); err != nil {
		return p.error(err.Error())
	}

	// TODO: remove DEBUG
	log.Printf("p.request.Parameter: %s", *p.request.Parameter)
	// log.Printf("p.request.Parameter: %+v", *p.request.ProtoFile[0])

	// TODO: how to parse and update Go file
	// https://golang.org/pkg/go/ast/#example_CommentMap
	// https://github.com/micro/protoc-gen-micro

	return p.error("not implemented")
}

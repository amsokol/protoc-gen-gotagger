package tagger

import (
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/fatih/structtag"
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
		in:          in,
		request:     &plugin_go.CodeGeneratorRequest{},
		targetFiles: map[string]goFile{},
		response:    &plugin_go.CodeGeneratorResponse{},
		out:         out,
	}
}

type plugin struct {
	in      io.Reader
	request *plugin_go.CodeGeneratorRequest

	xxxTag     *structtag.Tags
	outputPath string

	targetFiles map[string]goFile

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

func (p *plugin) parseParameter() error {
	var q bool
	params := strings.FieldsFunc(*p.request.Parameter, func(c rune) bool {
		if c == '"' {
			q = !q
		}
		return c == ',' && !q
	})

	re := regexp.MustCompile(`^(.+)=(.+)$`)
	for _, v := range params {
		m := re.FindStringSubmatch(v)
		if len(m) != 3 {
			return fmt.Errorf("failed to parse '%s' parameter: must be in 'key=value' format", v)
		}
		switch strings.ToLower(m[1]) {
		case "xxx":
			// we can't use ':' character in command parameter
			// so we use '+' instead and replace it by ':' after parsing
			var err error
			if p.xxxTag, err = structtag.Parse(strings.Replace(m[2], `+"`, `:"`, -1)); err != nil {
				return fmt.Errorf("failed to parse XXX tags '%s': %s", m[2], err.Error())
			}
		case "output_path":
			p.outputPath = m[2]
		default:
			return fmt.Errorf("unknown parameter: %s", m[1])
		}
	}

	return nil
}

func (p *plugin) Proccess() error {
	if err := p.read(); err != nil {
		return p.error(err.Error())
	}

	if err := p.parseParameter(); err != nil {
		return p.error("failed to parse 'gotagger_out' parameter value: %s", err.Error())
	}

	if err := p.analyzeSourceFiles(); err != nil {
		return p.error("failed to analyze source proto files: %s", err.Error())
	}

	if err := p.modifyTargetFiles(); err != nil {
		return p.error("failed to modify generated Go files: %s", err.Error())
	}

	return p.write()
}

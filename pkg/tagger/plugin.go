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

// Plugin is simple plugin interface.
// It has Process func only.
// It reads input stream data, proccesses files to add necessary tags and writes output
// according to the following specification:
// https://github.com/golang/protobuf/blob/master/protoc-gen-go/plugin/plugin.proto
type Plugin interface {
	// Proccess reads input stream data, proccesses files to add necessary tags and writes output.
	Proccess() error
}

// NewPlugin returns new object is implementing Plugin interface
// in - input stream that contains CodeGeneratorRequest serialized protop message
// out - output stream to store result CodeGeneratorResponse serialized proto message
func NewPlugin(in io.Reader, out io.Writer) Plugin {
	return &plugin{
		in:          in,
		request:     &plugin_go.CodeGeneratorRequest{},
		targetFiles: map[string]goFile{},
		response:    &plugin_go.CodeGeneratorResponse{},
		out:         out,
	}
}

// plugin is internal struct that contains funcs to process proto files and adds necessary tags to Go files
type plugin struct {
	// in is input stream that contains CodeGeneratorRequest serialized proto message
	in io.Reader

	// request is CodeGeneratorRequest proto message contains source proto files
	// See here for details: https://github.com/golang/protobuf/blob/master/protoc-gen-go/plugin/plugin.proto
	request *plugin_go.CodeGeneratorRequest

	// xxxTags are tags to add to the following fields for every struct:
	// XXX_NoUnkeyedLiteral
	// XXX_unrecognized
	// XXX_sizecache
	// Tags are provided in command line. Example:
	// protoc --proto_path=. -gotagger_out=xxx="bson+\"-\"",output_path=.:. data.proto
	xxxTags *structtag.Tags

	// outputPath is folder path where generated Go files are located
	// Example:
	// protoc --proto_path=. -gotagger_out=xxx="bson+\"-\"",output_path=./test:./test data.proto
	outputPath string

	// targetFiles is map (filename->content) is containing data to update Go files
	targetFiles map[string]goFile

	// response is CodeGeneratorResponse proto message contains updated Go files
	// See here for details: https://github.com/golang/protobuf/blob/master/protoc-gen-go/plugin/plugin.proto
	response *plugin_go.CodeGeneratorResponse

	// out is output stream to store CodeGeneratorResponse serialized proto message
	out io.Writer
}

// writeErrorResponse forms error message, put it in CodeGeneratorResponse message and stores error response in output stream
func (p *plugin) writeErrorResponse(err string, args ...interface{}) error {
	s := fmt.Sprintf(err, args...)
	p.response.Error = &s
	return p.writeResponse()
}

// readRequest reads CodeGeneratorRequest proto message from input stream to plugin.request field
func (p *plugin) readRequest() error {
	data, err := ioutil.ReadAll(p.in)
	if err != nil {
		return fmt.Errorf("failed to read marshaled request from input: %s", err.Error())
	}

	if err = proto.Unmarshal(data, p.request); err != nil {
		return fmt.Errorf("failed to unmarshal request from binary data: %s", err.Error())
	}

	// TODO: enable code bellow if you need to echo input data to stdin.bin file - it is used for debugging
	// ioutil.WriteFile("./stdin.bin", data, 0)

	return nil
}

// writeResponse writes plugin.response (CodeGeneratorResponse proto message) to output stream
func (p *plugin) writeResponse() error {
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

// parseParameter parse '-gotagger_out' command line option value
// It contains two comma delimited optional parameters:
// xxx - contains tags for XXX_NoUnkeyedLiteral, XXX_unrecognized, XXX_sizecache struct fields
// output_path - folder path where generated Go files are located
// Example:
// protoc --proto_path=. -gotagger_out=xxx="bson+\"-\"",output_path=./test:./test data.proto
func (p *plugin) parseParameter() error {
	var q bool
	params := strings.FieldsFunc(p.request.GetParameter(), func(c rune) bool {
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
			if p.xxxTags, err = structtag.Parse(strings.Replace(m[2], `+"`, `:"`, -1)); err != nil {
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

// Proccess is main and single public func of plugin that
// analyzes provided source proto files and returns updated Go files
func (p *plugin) Proccess() error {
	if err := p.readRequest(); err != nil {
		return p.writeErrorResponse(err.Error())
	}

	if err := p.parseParameter(); err != nil {
		return p.writeErrorResponse("failed to parse 'gotagger_out' parameter value: %s", err.Error())
	}

	if err := p.analyzeSourceFiles(); err != nil {
		return p.writeErrorResponse("failed to analyze source proto files: %s", err.Error())
	}

	if err := p.modifyTargetFiles(); err != nil {
		return p.writeErrorResponse("failed to modify generated Go files: %s", err.Error())
	}

	return p.writeResponse()
}

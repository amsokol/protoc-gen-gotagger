package cmd

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/amsokol/protoc-gen-gotagger/pkg/tagger"
)

// Run is tne plugin entrypoint.
// It may be run in debug mode if '--debug' parameter is provided.
// Debug mode ignores std input but reads input date from file is provided by --debug parameter.
// Example how to run in debug mode:
// protoc-gen-gotagger --debug=./stdin.bin
func Run() int {
	var debug string
	flag.StringVar(&debug, "debug", "", "debug input data file path")

	flag.Parse()

	var err error
	var in io.Reader
	if len(debug) > 0 {
		in, err = os.Open(debug)
		if err != nil {
			log.Printf("failed to open debug file '%s': %s", debug, err.Error())
			return 1
		}
	} else {
		in = os.Stdin
	}

	p := tagger.NewPlugin(in, os.Stdout)

	if err := p.Proccess(); err != nil {
		log.Print(err.Error())
		return 1
	}

	return 0
}

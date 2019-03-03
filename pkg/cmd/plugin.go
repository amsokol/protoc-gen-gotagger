package cmd

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/amsokol/protoc-gen-gotagger/pkg/tagger"
)

// Run is tne plugin entrypoint
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

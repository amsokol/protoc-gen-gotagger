package main

import (
	"os"

	"github.com/amsokol/protoc-gen-gotagger/pkg/cmd"
)

func main() {
	os.Exit(cmd.Run())
}

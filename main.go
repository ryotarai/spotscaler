package main

import (
	"github.com/ryotarai/spotscaler/cli"
	"os"
)

func main() {
	exitCode := cli.Run(os.Args[1:])
	os.Exit(exitCode)
}

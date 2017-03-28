package main

import (
	"os"

	"github.com/ryotarai/spotscaler/cli"
)

func main() {
	os.Exit(cli.Start(os.Args))
}

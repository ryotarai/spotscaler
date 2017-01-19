package main

import (
	"github.com/ryotarai/spotscaler/lib"
	"os"
)

func main() {
	os.Exit(autoscaler.StartCLI())
}

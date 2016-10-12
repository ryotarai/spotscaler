package main

import (
	"github.com/ryotarai/spot-autoscaler/lib"
	"os"
)

func main() {
	os.Exit(autoscaler.StartCLI())
}

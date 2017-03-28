package cli

import (
	"fmt"

	"github.com/ryotarai/spotscaler/spotscaler"
)

type versionCommand struct {
}

func (c *versionCommand) Help() string {
	return "Show version"
}

func (c *versionCommand) Synopsis() string {
	return "Show version"
}

func (c *versionCommand) Run(args []string) int {
	fmt.Printf("Spotscaler %s\n", spotscaler.Version)
	return 0
}

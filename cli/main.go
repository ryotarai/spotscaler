package cli

import (
	"fmt"
	"github.com/mitchellh/cli"
	"os"
)

// Run starts CLI
func Run(args []string) int {
	commands := Commands()

	cli := &cli.CLI{
		Args:     args,
		Commands: commands,
	}

	exitCode, err := cli.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing CLI: %s\n", err.Error())
		return 1
	}

	return exitCode
}

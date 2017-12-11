package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/ryotarai/spotscaler/scaler"
)

func Start(args []string) int {
	configPath := flag.String("config", "", "Config file")
	flag.Parse()
	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "-config option is not specified")
		return 1
	}

	c, err := scaler.NewConfigFromFile(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	s, err := scaler.NewScaler(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	s.Start()

	return 0
}

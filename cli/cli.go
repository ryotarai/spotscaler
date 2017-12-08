package cli

import "github.com/sirupsen/logrus"

func Start(args []string) int {
	logger := logrus.New()
	logger.Infof("Starting Spotscaler v%s", Version)

	return 0
}

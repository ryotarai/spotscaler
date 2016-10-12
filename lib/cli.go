package autoscaler

import (
	"flag"
	"fmt"
	"github.com/hashicorp/logutils"
	"log"
	"os"
)

func init() {
	// overwrite flags set by gin...
	log.SetFlags(log.LstdFlags)
}

// StartCLI is entrypoint and returns exit code
func StartCLI() int {
	configPath := flag.String("config", "", "config file")
	confirmBeforeAction := flag.Bool("confirm-before-action", false, "confirmation before important actions")
	server := flag.String("server", "", "start API server")
	version := flag.Bool("version", false, "show version")
	logLevel := flag.String("log-level", "DEBUG", "log level (one of DEBUG, INFO, WARN and ERROR)")
	flag.Parse()

	SetLogLevel(*logLevel)

	if *version {
		fmt.Printf("spot-autoscaler v%s (%v)\n", Version, GitCommit)
		return 0
	}

	if *configPath == "" {
		log.Println("[ERROR] -config option is required")
		return 1
	}

	config, err := LoadYAMLConfig(*configPath)
	if err != nil {
		log.Println(err)
		return 1
	}
	if *confirmBeforeAction {
		config.ConfirmBeforeAction = *confirmBeforeAction
	}

	err = config.Validate()
	if err != nil {
		log.Println(err)
		return 1
	}

	log.Printf("[DEBUG] loaded config: %+v", config)

	if *server == "" {
		runner, err := NewRunner(config)
		if err != nil {
			log.Println(err)
			return 1
		}

		err = runner.StartLoop()

		if err != nil {
			log.Println(err)
			return 1
		}
	} else {
		status := NewStatusStore(config.RedisHost, config.RedisKeyPrefix)
		api := NewAPIServer(status)
		api.Run(*server)
	}

	return 0
}

func SetLogLevel(level string) {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel(level),
		Writer:   os.Stdout,
	}
	log.SetOutput(filter)
}

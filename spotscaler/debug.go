// +build debug
package spotscaler

import (
	"fmt"
	"os"
)

var debugMode = (os.Getenv("DEBUG") != "")

func debugf(format string, a ...interface{}) {
	if debugMode {
		fmt.Printf(format, a...)
	}
}

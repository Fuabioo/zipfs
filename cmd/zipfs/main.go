package main

import (
	"github.com/Fuabioo/zipfs/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		// Execute handles printing and os.Exit internally
		_ = err
	}
}

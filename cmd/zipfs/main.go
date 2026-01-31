package main

import (
	"fmt"
	"os"
)

func main() {
	// TODO: Wire up CLI root command
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "zipfs: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Placeholder until internal/cli is implemented
	fmt.Println("zipfs - zip file virtual filesystem")
	return nil
}

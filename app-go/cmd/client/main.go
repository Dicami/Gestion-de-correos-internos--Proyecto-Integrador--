package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"maizap/internal/client"
	"maizap/internal/config"
)

func main() {
	cfg, err := config.LoadClientConfig(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	console := client.NewConsole(client.NewAPIClient(cfg), os.Stdin, os.Stdout)
	if err := console.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

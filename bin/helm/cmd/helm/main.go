package main

import (
	"os"

	"helm/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}

package main

import (
	"os"

	"github.com/mikaelmello/pingo/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

package main

import (
	"os"

	"github.com/pomdtr/sunbeam/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

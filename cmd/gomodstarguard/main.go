package main

import (
	"os"

	"github.com/harmoniemand/gomodstarguard/internal/cli"
)

func main() {
	os.Exit(cli.Run())
}

package main

import (
	"context"
	"os"

	"github.com/matthewmueller/jig/internal/cli"
	"github.com/matthewmueller/logs"
)

func main() {
	log := logs.Default()
	if err := cli.New(log).Parse(context.Background(), os.Args[1:]...); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

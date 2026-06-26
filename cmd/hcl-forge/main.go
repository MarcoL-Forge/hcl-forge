package main

import (
	"log"
	"os"

	"github.com/Marc0l95/hclforge/internal/cli"
)

func run(args []string) error {
	return cli.Run(args)
}

func main() {
	if err := run(os.Args); err != nil {
		log.Fatal(err)
	}
}

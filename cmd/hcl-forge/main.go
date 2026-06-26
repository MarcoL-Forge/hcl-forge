package main

import (
	"log"
	"os"

	"github.com/Marc0l95/hclforge/internal/cli"
)

func main() {
	if err := cli.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
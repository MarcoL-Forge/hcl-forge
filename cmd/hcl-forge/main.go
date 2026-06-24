package main

import (
	"fmt"
	"os"

	"github.com/Marc0l95/hclforge/internal/document"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: hcl-forge <file>")
		os.Exit(1)
	}

	raw, err := document.LoadFile(os.Args[1])
	if err != nil {
		fmt.Printf("load file: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(string(raw))
}

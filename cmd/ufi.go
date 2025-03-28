package main

import (
	"log"

	"github.com/sonyamoonglade/ufi/internal/parser"
)

func main() {
	if err := parser.Run(); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"github.com/knaka/binc/lib"
	"log"
	"os"
)

func main() {
	var err error
	err = lib.Main(os.Args)
	if err != nil {
		log.Fatalf("Error: %+v", err)
	}
}

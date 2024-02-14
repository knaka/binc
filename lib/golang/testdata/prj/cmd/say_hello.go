package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Println("Hello, World!!")
	log.Println("Hello, World!!")
	for _, arg := range os.Args {
		fmt.Println(arg)
	}
}

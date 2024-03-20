package main

import (
	"os"
	"strconv"

	"github.com/practigo/gomxf"
)

func main() {
	filename := os.Args[1]

	var (
		err error
		n   = -1
	)
	if len(os.Args) > 2 {
		n, err = strconv.Atoi(os.Args[2])
		if err != nil {
			panic(err)
		}
	}

	if err = gomxf.View(filename, n); err != nil {
		panic(err)
	}
}

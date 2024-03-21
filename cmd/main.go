package main

import (
	"flag"

	"github.com/practigo/gomxf"
)

var (
	n           = flag.Int("n", -1, "klv elements to read")
	showFill    = flag.Bool("f", false, "show Fill-Item")
	showUnKnown = flag.Bool("u", false, "show Unknown KLV")
)

func main() {
	flag.Parse()
	filename := flag.Arg(0)

	if err := gomxf.View(filename, &gomxf.Config{
		NRead:       *n,
		ShowUnKnown: *showUnKnown,
		ShowFill:    *showFill,
	}); err != nil {
		panic(err)
	}
}

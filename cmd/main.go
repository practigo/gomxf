package main

import (
	"flag"

	"github.com/practigo/gomxf"
)

var (
	n           = flag.Int("n", -1, "klv elements to read")
	level       = flag.Int("l", 1, "parse level")
	showUnKnown = flag.Bool("u", false, "show Unknown KLV (level 2+)")
	max         = flag.Int("m", 32, "maximum number of KLVs to show under a partition (level 2+)")
	koi         = flag.String("i", "", "KLV of interest to show in details, in format {part}:{idx}[:{style}]")
)

func main() {
	flag.Parse()
	filename := flag.Arg(0)

	if err := gomxf.Parse(filename, &gomxf.Config{
		NRead:       *n,
		Level:       *level,
		ShowUnKnown: *showUnKnown,
		Max:         *max,
		KOI:         *koi,
	}); err != nil {
		panic(err)
	}
}

package main

import (
	"flag"

	"github.com/practigo/gomxf"
)

var (
	n           = flag.Int("n", -1, "klv elements to read")
	showFill    = flag.Bool("f", false, "show Fill-Item")
	showUnKnown = flag.Bool("u", false, "show Unknown KLV")
	roi         = flag.Int("i", -1, "explore the #i KLV in details")
	showData    = flag.Bool("d", false, "show raw data fo $i KLV in []byte format")
	set         = flag.Bool("s", false, "try to parse the #i KLV as Local Sets (Metadata)")
)

func main() {
	flag.Parse()
	filename := flag.Arg(0)

	if err := gomxf.View(filename, &gomxf.Config{
		NRead:       *n,
		ShowUnKnown: *showUnKnown,
		ShowFill:    *showFill,
		ROI:         *roi,
		ShowRaw:     *showData,
		AsSets:      *set,
	}); err != nil {
		panic(err)
	}
}

package main

import(
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/skypies/geo"
)

var(
	fVerbosity int
)
	
func init() {
	flag.IntVar(&fVerbosity, "v", 0, "verbosity level")
	flag.Parse()
}


func main() {
	if len(flag.Args()) == 0 {
		log.Fatal("usage: fgeo 123.123, 123.123\n")
	}

	in := strings.Join(flag.Args(), " ")
	pos := geo.NewLatlong(in)

	fmt.Printf(">>>> %s\n  << (%.7f, %.7f)\n", in, pos.Lat, pos.Long)
	fmt.Printf("  << %T{Lat:%.7f, Long:%.7f}\n", pos, pos.Lat, pos.Long)
	fmt.Printf("  << {pos:{lat: %.7f , lng:  %.7f}},\n", pos.Lat, pos.Long)
}


// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

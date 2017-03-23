package main

import(
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/db"
)

var(
	ctx = context.Background()
	p = db.CloudDSProvider{"serfr0-fdb"}
	fVerbosity int
	fFoiaOnly bool
)
	
func init() {
	flag.IntVar(&fVerbosity, "v", 0, "verbosity level")
	flag.BoolVar(&fFoiaOnly, "foia", false, "FOIA data only")
	flag.Parse()
}

func main() {
	for _,arg := range flag.Args() {
		q := db.NewFlightQuery()

		if fFoiaOnly {
			q.ByTags([]string{"FOIA"})
		}

		if idspec,err := fdb.NewIdSpec(arg); err == nil {
			q.ByIdSpec(idspec)
		} else {
			//q.ByCallsign(arg)
		}
		
		cutoff,err := time.Parse("2006-01-02 15:04 -0700 MST", "2017-01-01 04:00 -0700 PDT")
		if false && err == nil  {
			q.Filter("LastUpdate > ", cutoff).Limit(100)
		}
		fmt.Printf("Running query for '%s': %s", arg, q)

		flights,err := db.GetAllByQuery(ctx, p, q)
		if err != nil { log.Fatal(err) }

		for i,f := range flights {
			s,_ := f.Times()
			str := fmt.Sprintf("%25.25s trk=%s upd=%s %v", f.IdentityString(), s, f.GetLastUpdate(),
				f.TagList())
			fmt.Printf("[%2d] %s\n", i, str)
		}
		fmt.Printf("\n")
		if fVerbosity > 0 {
			for i,f := range flights {
				str := fmt.Sprintf("----{ %d : %s }----\n", i, f.IdentityString())
				str += fmt.Sprintf("    %s\n", f.IdSpec())
				str += fmt.Sprintf("    %s\n", f.FullString())
				str += fmt.Sprintf("    airframe: %s\n", f.Airframe.String())
				str += fmt.Sprintf("    %s\n\n", f)
				str += fmt.Sprintf("    index tags: %v\n", f.IndexTagList())
				str += fmt.Sprintf("    /batch/flights/flight?flightkey=%s&job=retag\n", f.GetDatastoreKey())

				t := f.AnyTrack()
				str += fmt.Sprintf("\n-- Anytrack: %s\n", t)
			
				for k,v := range f.Tracks {
					str += fmt.Sprintf("  -- [%-7.7s] %s\n", k, v)
					if false {
						for n,tp := range *v {
							str += fmt.Sprintf("    - [%3d] %s\n", n, tp)
						}
					}
				}
				str += "\n"
				if fVerbosity > 1 {
					str += fmt.Sprintf("---- DebugLog:-\n%s\n", f.DebugLog)
				}
				fmt.Print(str)
			}
		}
	}
}


// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

package main

import(
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/skypies/adsb"
	"github.com/skypies/util/date"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/db"
)

var(
	ctx = context.Background()
	p = db.CloudDSProvider{"serfr0-fdb"}
	fVerbosity int
	fFoiaOnly bool
	fInPdt bool
	fLimit int
	fIcaoId string
	fCallsign string
)
	
func init() {
	flag.IntVar(&fVerbosity, "v", 0, "verbosity level")
	flag.BoolVar(&fFoiaOnly, "foia", false, "FOIA data only")
	flag.BoolVar(&fInPdt, "pdt", true, "show timestamps in PDT")
	flag.IntVar(&fLimit, "limit", 10, "how many matches to retrieve")
	flag.StringVar(&fIcaoId, "icao", "", "ICAO id for airframe (6-digit hex)")
	flag.StringVar(&fCallsign, "callsign", "", "Callsign, or maybe registration, for a flight")
	flag.Parse()
}

// Based on the various command line flags
func queryFromArgs() *db.Query {
	q := db.NewFlightQuery()
	q.Limit(fLimit)
	if fFoiaOnly {q.ByTags([]string{"FOIA"}) }
		// last updated stuff
		//cutoff,err := time.Parse("2006-01-02 15:04 -0700 MST", "2017-01-01 04:00 -0700 PDT")
		//if false && err == nil  {
		//	q.Filter("LastUpdate > ", cutoff).Limit(100)
		//}

	if fIcaoId != "" { q.ByIcaoId(adsb.IcaoId(fIcaoId)) }
	if fCallsign != "" { q.ByCallsign(fCallsign) }

	q.Order("-LastUpdate")
	
	return q
}

func runQuery(q *db.Query) {
	fmt.Printf("Running query %s", q)

	flights,err := db.GetAllByQuery(ctx, p, q)
	if err != nil { log.Fatal(err) }

	for i,f := range flights {
		s,_ := f.Times()
		if fInPdt { s = date.InPdt(s) }

		n := len(f.AnyTrack())
		str := fmt.Sprintf("%25.25s %s %4dpts %s", f.IdentityString(), s, n, f.IdSpecString())

		fmt.Printf("[%2d] %s\n", i, str)
	}
	fmt.Printf("\n")

	if fVerbosity > 0 {
		for i,f := range flights {
			str := fmt.Sprintf("----{ %d : %s }----\n", i, f.IdentityString())
			str += fmt.Sprintf("    idspec: %s    key %s\n", f.IdSpec(), f.GetDatastoreKey())
			str += fmt.Sprintf("    airframe: %s\n", f.Airframe.String())
			str += fmt.Sprintf("    index tags: %v\n", f.IndexTagList())
			str += fmt.Sprintf("    - Anytrack: %s\n", f.AnyTrack())
			
			for k,v := range f.Tracks {
				str += fmt.Sprintf("    - [%-7.7s] %s\n", k, v)
				if fVerbosity > 2 {
					for n,tp := range *v {
						str += fmt.Sprintf("      - [%3d] %s\n", n, tp)
					}
				}
			}
			for _,t := range f.Timeslots() {
				str += fmt.Sprintf("    ** timeslot: [%s] %s\n", t, date.InPdt(t))
			}
			
			if fVerbosity > 1 {
				str += fmt.Sprintf("---- DebugLog:-\n%s\n", f.DebugLog)
			}
			str += "\n"
			fmt.Print(str)
		}
	}
}

func main() {
	if len(flag.Args()) == 0 {
		runQuery(queryFromArgs())
	}

	// assume it's all idspecs ...
	for _,arg := range flag.Args() {
		if idspec,err := fdb.NewIdSpec(arg); err == nil {
			fmt.Printf("Idspec time: %s (%s)\n", idspec.Time, date.InPdt(idspec.Time))
			runQuery(queryFromArgs().ByIdSpec(idspec))
		} else {
			log.Fatal("bad idspec '%s': %v\n", arg, err)
		}
	}
}


// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

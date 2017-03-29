package main

import(
	"compress/gzip"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	
	"golang.org/x/net/context"

	"github.com/skypies/geo"
	"github.com/skypies/util/date"
	"github.com/skypies/util/dsprovider"
	"github.com/skypies/util/histogram"
	
	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/faadata"
)

var(
	ctx = context.Background()
	p = dsprovider.CloudDSProvider{"serfr0-fdb"}
	fDryRun bool
	fCmd    string
)
func init() {
	flag.BoolVar(&fDryRun, "dryrun", true, "in dryrun mode, don't change the database")
	flag.StringVar(&fCmd, "cmd", "stats", "what to do: {stats}")
	flag.Parse()
}

// {{{ loadfile

func loadfile(file string, callback faadata.NewFlightCallback) {
	if rdr, err := os.Open(file); err != nil {
		log.Fatal("open '%s': %v\n", file, err)
	} else if gzRdr,err := gzip.NewReader(rdr); err != nil {
		log.Fatal("gzopen '%s': %v\n", file, err)
	} else if n,str,err := faadata.ReadFrom(ctx, file, gzRdr, callback); err != nil {
		log.Fatal("faadata.ReadFrom '%s': %v\n", file, err)
	} else {
		_,_ = n,str
		//fmt.Printf("Completed, %d said true, here is aggregate out:-\n%s", n, str)
	}
}

// }}}
// {{{ stats

// {{{ pprint

func pprint(m map[string]int) string {
	str := ""
	keys := []string{}
	for k,_ := range m { keys = append(keys, k ) }
	sort.Strings(keys)
	small := 0
	for _,k := range keys {
		if m[k] < 10 {
			small += m[k]
			continue
		}
		str += fmt.Sprintf("  %-12.12s:  %5d\n", k, m[k])
	}
	if small > 0 {
		str += fmt.Sprintf("  %-12.12s:  %5d\n", "{smalls}", small)
	}
	return str
}

// }}}

func stats(files []string) {
	norcal := map[string]int{}
	icao := map[string]int{}
	h := histogram.NewSet(1000)
	tod := histogram.Histogram{NumBuckets:48,ValMax:48}
	var bbox *geo.LatlongBox
	
	callback := func(ctx context.Context, f *fdb.Flight) (bool, string, error) {
		for _,tag := range []string{":SFO","SFO:",":SJC","SJC:",":OAK","OAK:"} {
			if f.HasTag(tag) { norcal[tag]++ }
		}
		icao[f.Schedule.ICAO]++
		t := *f.Tracks["FOIA"]
		h.RecordValue("tracklen", int64(len(t)))
		if bbox == nil {
			tmp := t[0].BoxTo(t[1].Latlong)
			bbox = &tmp
		}
		for _,tp := range t {
			bbox.Enclose(tp.Latlong)
			// Figure out which 30m bucket this data is from
			hr := date.InPdt(tp.TimestampUTC).Hour()
			m := date.InPdt(tp.TimestampUTC).Minute()
			bucket := hr*2
			if m>=30 { bucket++ }
			tod.Add(histogram.ScalarVal(bucket))
		}

		return false,"",nil
	}

	for i,file := range files {
		fmt.Printf("[%d/%d] loading %s\n", i+1, len(files), file)
		loadfile(file, callback)
	}

	wd,ht := bbox.NW().DistKM(bbox.NE), bbox.NW().DistKM(bbox.SW)
	
	fmt.Printf("Area (%.1fKM x %.1fKM) : %s\n", wd, ht, *bbox)
	fmt.Printf("  <http://fdb.serfr1.org/fdb/map?boxes=b1&"+bbox.ToCGIArgs("b1")+">\n")
	fmt.Printf("Airports:-\n%s", pprint(norcal))
	fmt.Printf("ICAO codes:-\n%s", pprint(icao))
	fmt.Printf("Time of day counts: %s\n", tod)
	fmt.Printf("Stats:-\n%s", h)
}

// }}}

func main() {
	switch fCmd {
	case "stats": stats(flag.Args())
	default: log.Fatal("command '%s' not known", fCmd)
	}
}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

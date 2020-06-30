package main

// usage: tail -f ~/swim/log/messages.log | swim

import(
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"golang.org/x/net/context"

	"github.com/skypies/adsb"
	"github.com/skypies/pi/airspace"
	"github.com/skypies/util/date"
	"github.com/skypies/util/gcp/ds"
	"github.com/skypies/util/gcp/singleton"

	"github.com/skypies/flightdb/swim"
)

var(
	ProjectName   = "serfr0-fdb"
	SingletonName = "swim-airspace" // to identify the datastore singleton entity
	ctx           = context.Background()
	dumpDir       = "" // "/home/abw/msgs"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	max := 400000
	scanner.Buffer(make([]byte, max), max)

	p,err := ds.NewCloudDSProvider(ctx, ProjectName)
	if err != nil { log.Fatal(err) }

	
	as := airspace.NewAirspace()

	tLastFlush := time.Now()
	
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

		txt := scanner.Text()
		
		if dumpDir != "" {
			fname := fmt.Sprintf("%s/msg-%s", dumpDir, date.NowInPdt().Format("20060102-150405"))
			jsonMap := map[string]interface{}{}
			json.Unmarshal([]byte(txt), &jsonMap)
			contents,_ := json.MarshalIndent(jsonMap, "", "  ")
			ioutil.WriteFile(fname, []byte(contents), 0644)
		}			
			
		for _,f := range swim.Json2Flights(txt) {
			if dumpDir != "" {
				fname := fmt.Sprintf("%s/flight-%s-%s", dumpDir, date.NowInPdt().Format("20060102-150405"), f.Source)
				contents,_ := json.MarshalIndent(f, "", "  ")
				ioutil.WriteFile(fname, []byte(contents), 0644)
			}

			if f.Source != "TH" { continue }

			msg := f.AsAdsb()
			as.MaybeUpdate([]*adsb.CompositeMsg{&msg})
			log.Printf("  ** %s\n", msg)
		}

		if time.Since(tLastFlush) > time.Second {
			tLastFlush = time.Now()
			flushToDatastore(p, as)
		}
	}
}

func flushToDatastore(p ds.DatastoreProvider, as airspace.Airspace) {
	log.Printf("**** flushing airspace ****\n%s", as)
	sp := singleton.NewProvider(p)
	if err := sp.WriteSingleton(ctx, SingletonName, nil, &as); err != nil {
		log.Printf("sp.WriteSingleton(%s) err: %v\n", SingletonName, err)
	}
}

package main

import(
	"fmt"
	"net/http"
	"time"
	
	"google.golang.org/appengine"
	"google.golang.org/appengine/memcache"
	"google.golang.org/appengine/urlfetch"
	"golang.org/x/net/context"

	"github.com/skypies/geo/sfo"
	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fr24"
	"github.com/skypies/flightdb2/ref"
)
// dev_appserver.py --clear_datastore=yes ./ui.yaml

func init() {
	http.HandleFunc("/fdb/fr24", fr24PollHandler)
	http.HandleFunc("/fdb/fr24q", fr24QueryHandler)
}

// {{{ listResult2Airframe

func listResult2Airframe(fs fdb.FlightSnapshot) fdb.Airframe {
	af := fdb.Airframe{
		Icao24:fs.IcaoId,
		Registration:fs.Airframe.Registration,
		EquipmentType:fs.Airframe.EquipmentType,
	}

	c := fdb.NewCallsign(fs.Callsign)
	if c.CallsignType == fdb.IcaoFlightNumber {
		af.CallsignPrefix = c.IcaoPrefix
	}
	
	return af
}

// }}}
// {{{ updateAirframeCache

func updateAirframeCache(c context.Context, resp []fdb.FlightSnapshot, list bool) (string,error) {
	airframes := ref.NewAirframeCache(c)
	str := ""
	n := 0

	for _,fs := range resp {
		if fs.Airframe.Registration == "" { continue }
		newAf := listResult2Airframe(fs)
		oldAf := airframes.Get(fs.IcaoId)

		if oldAf == nil || oldAf.CallsignPrefix != newAf.CallsignPrefix {
			airframes.Set(&newAf)
			str += fmt.Sprintf("* [%7.7s]%s %s\n", fs.Airframe.Registration, newAf, fs)
			n++
		}
	}

	if n>0 {
		if err := airframes.Persist(c); err != nil {
			return "[error]", err
		}
	}

	if list {
		return str + fmt.Sprintf("\n%s\n", airframes), nil
	} else {
		return str + fmt.Sprintf("\nhave %d airframes stored\n", len(airframes.Map)), nil
	}
}

// }}}

// {{{ {load,save}FIFOSet

var kFIFOSetMaxAgeMins = 120

// This is stored in memcache, so it can vanish
func loadFIFOSet(c context.Context, set *fdb.FIFOSet) (error) {
	if _,err := memcache.Gob.Get(c, fdb.KMemcacheFIFOSetKey, set); err == memcache.ErrCacheMiss {
    // cache miss, but we don't care
		return nil
	} else if err != nil {
    //c.Errorf("error getting item: %v", err)
		return err
	}
	return nil
}

func saveFIFOSet(c context.Context, s fdb.FIFOSet) error {
	s.AgeOut(time.Minute * time.Duration(kFIFOSetMaxAgeMins))
	item := memcache.Item{Key:fdb.KMemcacheFIFOSetKey, Object:s}
	return memcache.Gob.Set(c, &item)
}

// }}}

// {{{ fr24PollHandler

// Need a grid of which fr24 data fields are provided by which call.

func fr24PollHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	fr,_ := fr24.NewFr24(urlfetch.Client(c))
	// db := fgae.FlightDB{C:c}

	flights,err := fr.LookupCurrentList(sfo.KLatlongSFO.Box(160,160))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	str,err := updateAirframeCache(c, flights, r.FormValue("list") != "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Are any of these new ? use {callsign,reg} as the unique ID (unique within NN hours, anyhow)
	set := fdb.FIFOSet{}
	if err := loadFIFOSet(c,&set); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	new := set.FindNew(flights)
	if err := saveFIFOSet(c,set); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	/* LookupHistory is an expensive way to get from any two of
     {callsign,depaturedate,registration} to a fr24ID, the thing that
     lets us lookup track data. Shame.

     It also gives us robust arrival times, for accurate taskqueue scheduling. */
	for _,fs := range new {		

		_=fs
	}
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\n\n%s", str)))
}

// }}}
// {{{ fr24QueryHandler

func fr24QueryHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	fr,_ := fr24.NewFr24(urlfetch.Client(c))

	id,err := fr.LookupQuery(r.FormValue("q"))
	if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
	}
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\nq=%s\n%#v\n", r.FormValue("q"), id)))
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

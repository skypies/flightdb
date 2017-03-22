package backend

import(
	"fmt"
	"net/http"
	"time"
	
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/memcache"
	"google.golang.org/appengine/urlfetch"
	"golang.org/x/net/context"

	"github.com/skypies/geo/sfo"
	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/fgae"
	"github.com/skypies/flightdb/fr24"
	"github.com/skypies/flightdb/ref"
)
// dev_appserver.py --clear_datastore=yes ./ui.yaml

func init() {
	http.HandleFunc("/be/fr24", fr24PollHandler)
	http.HandleFunc("/be/fr24q", fr24QueryHandler)
	http.HandleFunc("/be/schedcache/view", schedcacheViewHandler)
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
// {{{ updateScheduleCache

func updateScheduleCache(c context.Context, resp []fdb.FlightSnapshot) error {

	if len(resp) == 0 { return nil } // Don't overwrite in cases of error
	
	sc := ref.BlankScheduleCache()
	for i,_ := range resp {
		sc.Map[resp[i].IcaoId] = &resp[i]
	}
	sc.LastUpdated = time.Now()
	
	if err := sc.Persist(c); err != nil {
		log.Errorf(c, "updateScheduleCache/Persist: %v", err)
		return err
	}

	return nil
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
	db := fgae.NewDBFromReq(r)
	fr,_ := fr24.NewFr24(db.HTTPClient())

	db.Perff("fr24Poll_100", "making call")
	flights,err := fr.LookupCurrentList(sfo.KLatlongSFO.Box(320,320))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	db.Perff("fr24Poll_150", "call returned (%d flights), updating airframes", len(flights))

	str,err := updateAirframeCache(db.Ctx(), flights, r.FormValue("list") != "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	db.Perff("fr24Poll_200", "updating schedulecache")
	if err := updateScheduleCache(db.Ctx(), flights); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Load up the list of things we've acted on 'recently'
	db.Perff("fr24Poll_250", "loading alreadyProcessed fifoset")
	alreadyProcessed := fdb.FIFOSet{}
	if err := loadFIFOSet(db.Ctx(),&alreadyProcessed); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	db.Perff("fr24Poll_250", "fifoset loaded (%d entries); calling FindNew()", len(alreadyProcessed))

	// Filter out the things we've acted on recently (and, speculatively, add them
	// to the 'processed' list
	new := alreadyProcessed.FindNew(flights)
	str += fmt.Sprintf("\n----{ %d flights, %d new, set=%d }----\n", len(flights), len(new),
		len(alreadyProcessed))
	db.Perff("fr24Poll_400", "FindNew found %d new flights (set now=%d). Iterating!", len(new),
		len(alreadyProcessed))
	
	// Review the flights we've yet to process.
	tstamp := time.Now().Add(-30 * time.Second)
	nMerges,nUpdates := 0,0
	for _,fs := range new {
		if fs.IcaoId == "" {
			continue
		}

		idspec := fdb.IdSpec{IcaoId:fs.IcaoId, Time:tstamp}
		f,err := db.LookupMostRecent(db.NewQuery().ByIdSpec(idspec))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if f == nil {
			// This flight hasn't shown up yet. So remove it from the 'seen' set.
			alreadyProcessed.Remove(fs)
			continue
		}

		// OK ! Maybe update our DB with schedule data ...
		str2 := fmt.Sprintf("-- %s\n - %s:%s %v\n", fs, f.IcaoId, f.FullString(), f.TagList())
		nMerges++
		if changed := f.MergeIdentityFrom(fs.Flight); changed == true {
			f.Analyse()
			str += str2 + fmt.Sprintf(" - %s:%s %v\n", f.IcaoId, f.FullString(), f.TagList())

			nUpdates++
			if err := db.PersistFlight(f); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	db.Perff("fr24Poll_450", "Iteration done. %d merges, %d updates", nMerges, nUpdates)
	
	alreadyProcessed.AgeOut(2 * time.Hour)
	if err := saveFIFOSet(db.Ctx(),alreadyProcessed); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	db.Perff("fr24Poll_500", "fifoset saved, all done")
	
	/* FWIW ... LookupHistory is an expensive way to get from any two of
     {callsign,depaturedate,registration} to a fr24ID, the thing that
     lets us lookup track data. Shame.

     It also gives us robust arrival times, for accurate taskqueue scheduling. */
	
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
// {{{ schedcacheViewHandler

func schedcacheViewHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	sc := ref.NewScheduleCache(c)
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\n%s\n", sc)))
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

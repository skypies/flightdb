// Use a shared workqueue ('batch') to do some processing against the entire database.
package fgae

// http://fdb.serfr1.org/batch/flights/dates?job=retag&date=range&range_from=2014/01/01&range_to=2015/12/31&tags=FOIA

// http://fdb.serfr1.org/batch/flights/dates?job=retag&date=yesterday&tags=:SFO

import (
	"fmt"
	"net/http"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"

	"github.com/skypies/util/date"
	"github.com/skypies/util/widget"

	adsblib "github.com/skypies/adsb"
	fdb "github.com/skypies/flightdb"
)

var(
	QueueName   = "batch"
	RangeUrl    = "/batch/flights/dates"
	DayUrl      = "/batch/flights/day"
	InstanceUrl = "/batch/flights/flight"
)

// {{{ formValueFlightByKey

// A super widget
func formValueFlightByKey(r *http.Request) (*fdb.Flight, error) {
	ctx,_ := context.WithTimeout(appengine.NewContext(r), 10*time.Minute)
	db := NewDB(ctx)

	keyer,err := db.Backend.DecodeKey(r.FormValue("flightkey"))
	if err != nil {
		return nil, err
	}

	return db.LookupKey(keyer)
}

// }}}

// {{{ BatchFlightDateRangeHandler

// /batch/flights/dates?
//   &job=retag
//   &tags=FOO,BAR
//   &date=range&range_from=2016/01/21&range_to=2016/01/26
//   &dryrun=1

// Enqueues one 'day' task per day in the range
func BatchFlightDateRangeHandler(w http.ResponseWriter, r *http.Request) {
	c,_ := context.WithTimeout(appengine.NewContext(r), 10*time.Minute)

	n := 0
	str := ""
	s,e,_ := widget.FormValueDateRange(r)
	job := r.FormValue("job")
	if job == "" {
		http.Error(w, "Missing argument: &job=foo", http.StatusInternalServerError)
		return
	}
	
	str += fmt.Sprintf("** s: %s\n** e: %s\n", s, e)

	tags := r.FormValue("tags")
	days := date.IntermediateMidnights(s.Add(-1 * time.Second),e) // decrement start, to include it
	for _,day := range days {

		dayStr := day.Format("2006/01/02")
		
		str += fmt.Sprintf(" * adding %s tags=%v, %s via %s\n", job, tags, dayStr, DayUrl)
		
		if r.FormValue("dryrun") == "" {
			// TODO: pass through *all* the CGI args
			t := taskqueue.NewPOSTTask(DayUrl, map[string][]string{
				"day": {dayStr},
				"job": {job},
				"tags": {tags},
			})

			if _,err := taskqueue.Add(c, t, QueueName); err != nil {
				log.Errorf(c, "upgradeHandler: enqueue to %s: %v", QueueName, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			n++
		}
	}

	log.Infof(c, "enqueued %d batch days for '%s'", n, job)

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK batch, enqueued %d day tasks for %s\n%s", n, job, str)))
}

// }}}
// {{{ BatchFlightDayHandler

// /batch/flights/day?
//   &day=2016/01/21
//   &job=foo
//   &tags=FOO,BAR

// Dequeue a single day, and enqueue a job for each flight on that day
func BatchFlightDayHandler(w http.ResponseWriter, r *http.Request) {
	ctx,_ := context.WithTimeout(appengine.NewContext(r), 10*time.Minute)
	db := NewDB(ctx)

	job := r.FormValue("job")
	if job == "" {
		http.Error(w, "Missing argument: &job=foo", http.StatusInternalServerError)
	}

	tStart := time.Now()
	tags := widget.FormValueCommaSepStrings(r, "tags")
	day := date.ArbitraryDatestring2MidnightPdt(r.FormValue("day"), "2006/01/02")
	start,end := date.WindowForTime(day)
	end = end.Add(-1 * time.Second)
	
	q := QueryForTimeRange(tags,start,end)
	keyers,err := db.LookupAllKeys(q)
	if err != nil {
		errStr := fmt.Sprintf("elapsed=%s; err=%v", time.Since(tStart), err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}	

	str := fmt.Sprintf("* start: %s\n* end  : %s\n* tags : %q\n* n    : %d\n",
		start,end,tags,len(keyers))

	n := 0
	for i,keyer := range keyers {
		if i<10 { str += " "+InstanceUrl+"?job="+job+"flightkey="+keyer.Encode() + "\n" }
		n++

		t := taskqueue.NewPOSTTask(InstanceUrl, map[string][]string{
			"flightkey": {keyer.Encode()},
			"job": {job},
		})
		if _,err := taskqueue.Add(ctx, t, QueueName); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		n++
	}

	log.Infof(ctx, "enqueued %d batch items for '%s'", n, job)

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK, batch, enqueued %d tasks for %s\n%s", n, job, str)))
}

// }}}
// {{{ BatchFlightHandler

// To run a job directly: /batch/flights/flight?job=retag&flightkey=...&
func BatchFlightHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	f,err := formValueFlightByKey(r)
	if err != nil {
		log.Errorf(c, "/batch/fdb/track/getflight: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// You now have a job name, and a flight object. Get to it !	
	job := r.FormValue("job")
	str := ""
	switch job {
	case "retag":         str,err = jobRetagHandler(r,f)
	case "breakup":       str,err = jobMaybeBreakupFlight(r,f)
	}

	if err != nil {
		log.Errorf(c, "%s", str)
		log.Errorf(c, "/backend/fdb/batch/flight: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Debugf(c, "job=%s, on %s: %s", job, f, str)
		
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\n* job: %s\n* %s\n%s\n", job, f, str)))
}

// }}}

// {{{ jobRetagHandler

// Note; this never removes any tags. It only adds new ones.
func jobRetagHandler(r *http.Request, f *fdb.Flight) (string, error) {
	c := appengine.NewContext(r)

	str := fmt.Sprintf("OK\nbatch, for [%s]\n", f)
	
	str += fmt.Sprintf("\n* Pre WP: %v\n", f.WaypointList())
	str += fmt.Sprintf("* Pre Tags: %v\n", f.TagList())
	str += fmt.Sprintf("* Pre IndexingTags: %v\n", f.IndexTagList())

	f.Analyse()

	str += fmt.Sprintf("\n* Post WP: %v\n", f.WaypointList())
	str += fmt.Sprintf("* Post Tags: %v\n", f.TagList())
	str += fmt.Sprintf("* Post IndexingTags: %v\n", f.IndexTagList())

	str += fmt.Sprintf("\n*** URL: /fdb/tracks?idspec=%s\n", f.IdSpecString())
	
	if true {
		db := NewDB(c)
		if err := db.PersistFlight(f); err != nil {
			str += fmt.Sprintf("* Failed, with: %v\n", err)	
			db.Errorf("%s", str)
			return str, err
		}
		db.Infof("%s", str)
	}	

	return str, nil
}

// }}}
// {{{ jobMaybeBreakupFlight

// There was a nasty bug, where a new flight could be erroneously
// glued onto the previous flight if the track type was different
// (e.g. ADSB vs MLAT), because it took 'absence of this track type'
// to mean that the flight was empty. That's now fixed, but lots of
// bad data exists in the database. This oneoff cleanup routine
// separates these flights out, either into the 'real' flight it
// should be in, or a new one.

func jobMaybeBreakupFlight(r *http.Request, f *fdb.Flight) (string, error) {
	db := NewDB(appengine.NewContext(r))

	mlat,adsb := f.Tracks["MLAT"],f.Tracks["ADSB"]
	if adsb == nil {
		return "No ADSB\n", nil
	} else if mlat == nil {
		return "No MLAT\n", nil
	}

	// Order the two tracks based on their end-time
	older,newer,trackKey := mlat,adsb,"ADSB"
	if (*mlat).End().After( (*adsb).End() ) {
		older,newer,trackKey = adsb,mlat,"MLAT"
	}

	plausible,debstr := older.PlausibleExtension(newer)
	str := fmt.Sprintf("has both ! plausible=%v\n%s\n", plausible, debstr)

	if plausible {
		return str+"\nPLAUSIBLE, leave as is\n", nil
	}

	// Not plausible. Need to find a new home for 'newer'.
	
	// We can build an idpsec from the track's timestamp.
	idspec := f.IdSpec()
	idspec.Time = newer.End()
	str += fmt.Sprintf("** OK, looking to breakup\n\n\n")
	str += fmt.Sprintf("** http://fdb.serfr1.org/fdb/debug2?idspec=%s\n", idspec)

	// Look it up, iterate over results.
	results,err := db.LookupAll(db.NewQuery().ByIdSpec(idspec))
	if err != nil { return str, err }

	str += fmt.Sprintf("\n=== %s\n", f.IdentityString())
	for k,v := range f.Tracks {
		str += fmt.Sprintf(" == [%-7.7s] %s\n", k, v)
	}
	str += fmt.Sprintf(" ++ [       ] %s (TRYING TO MOVE)\n", newer)
	
	haveFoundNewHome := false
	for _,maybeF := range results {
		if f.GetDatastoreKey() == maybeF.GetDatastoreKey() {
			str += fmt.Sprintf("\n--- %s [SKIP, this is src flight]\n", maybeF.IdentityString())
			continue
		}
		str += fmt.Sprintf("\n--- %s\n", maybeF.IdentityString())

		// Look for the the flight whose existing track is a plausible extension (or vice versa?)
		for k,v := range maybeF.Tracks {
			str += fmt.Sprintf(" -- [%-7.7s] %s\n", k, v)

			p1,str1 := v.PlausibleExtension(newer)
			p2,str2 := newer.PlausibleExtension(v)
			if p1 {
				str += fmt.Sprintf("  - PLAUSIBLE EXTENSION 1, Yahay !\n%s\n", str1)
			} else if p2 {
				str += fmt.Sprintf("  - PLAUSIBLE EXTENSION 2, Yahay !\n%s\n", str2)
			}

			// if we find one, WOWZER; add/move it over, and save both; else flag error and leave alone.
			if p1 || p2 {
				if !maybeF.HasTrack(trackKey) {
					maybeF.Tracks[trackKey] = &fdb.Track{}
				}
				maybeF.Tracks[trackKey].Merge(newer)
				delete(f.Tracks, trackKey)

				f.Analyse()
				maybeF.Analyse()

				str += fmt.Sprintf("\n\n================{ new stuff }===========================\n\n")
				str += fmt.Sprintf("\n--- %s\n", f.IdentityString())
				for k,v := range f.Tracks {
					str += fmt.Sprintf(" -- [%-7.7s] %s\n", k, v)
				}
				str += fmt.Sprintf("\n+++ %s\n", maybeF.IdentityString())
				for k,v := range maybeF.Tracks {
					str += fmt.Sprintf(" ++ [%-7.7s] %s\n", k, v)
				}
				
				if err := db.PersistFlight(f); err != nil {
					str += fmt.Sprintf("* Failed1, with: %v\n", err)	
					db.Errorf("%s", str)
					return str, err
				}
				if err := db.PersistFlight(maybeF); err != nil {
					str += fmt.Sprintf("* Failed2, with: %v\n", err)	
					db.Errorf("%s", str)
					return str, err
				}

				haveFoundNewHome = true
				break
			}
		}
	}

	if !haveFoundNewHome {
		str += "\n\nNo new home found :( Should create a whole new flight for newer\n"
		frag := &fdb.TrackFragment{
			IcaoId: adsblib.IcaoId(f.Identity.IcaoId),
			Callsign: f.Identity.Callsign,
			Track: *newer,
			DataSystem: fdb.DSADSB,
		}
		if trackKey == "MLAT" { frag.DataSystem = fdb.DSMLAT }

		newF := fdb.NewFlightFromTrackFragment(frag)

		str += "\n\nNo new home found :( Should create a whole new flight for newer\n"
		str += fmt.Sprintf("** Done !\n** NewF: [%s] %s\n", newF.IdentityString(), newF)

		delete(f.Tracks, trackKey)

		f.Analyse()
		newF.Analyse()
		
		if err := db.PersistFlight(f); err != nil {
			str += fmt.Sprintf("* Failed3, with: %v\n", err)	
			db.Errorf("%s", str)
			return str, err
		}
		if err := db.PersistFlight(newF); err != nil {
			str += fmt.Sprintf("* Failed4, with: %v\n", err)	
			db.Errorf("%s", str)
			return str, err
		}
	}
	
	return str, nil
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

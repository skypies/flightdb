package ui

import(
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine/user"
	"google.golang.org/appengine/urlfetch"

	"github.com/skypies/adsb"
	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/widget"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/fgae"
	"github.com/skypies/flightdb/fr24"
)

func init() {
	//http.HandleFunc("/fdb/debug/frags", UIOptionsHandler(debugFragsHandler))
	//http.HandleFunc("/fdb/debug/user", UIOptionsHandler(debugUserHandler))
}

// {{{ debugUserHandler

func debugUserHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	user := user.Current(ctx)
	json,_ := json.MarshalIndent(user, "", "  ")

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("OK\n--\n"+string(json)))
}

// }}}
// {{{ DebugHandler

func DebugHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	db := fgae.NewDB(ctx)
	opt,_ := GetUIOptions(ctx)
	str := ""

	/*
	conts,_ := ioutil.ReadDir(".")
	for _,f := range conts {
		str += fmt.Sprintf("Yoho: %s\n", f.Name())
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\n\n%s", str)))
	return
*/
	
	//str += fmt.Sprintf("** Idspecs:-\n%#v\n\n", opt.IdSpecs())

	idspecs,err := opt.IdSpecs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nPts := 0
	for _,idspec := range idspecs {
		q := db.NewQuery().ByIdSpec(idspec)
		str += fmt.Sprintf("*** %s [%v]\n%s\nResults:-\n", idspec, idspec, q)

		results,err := db.LookupAll(q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		for i,result := range results {
			//s,e := result.Times()
			//str += fmt.Sprintf("  * [%02d] %s,%s  %s\n", i, s,e, result.IdentityString())
			str += fmt.Sprintf(" [%02d] %s\n", i, result)
			nPts += len(result.AnyTrack())
		}
		str += fmt.Sprintf("*** Total pts: %d\n", nPts)
		str += "\n\n\n\n"
		
		for i,f := range results {
			str += fmt.Sprintf("----------{result %02d }-----------\n\n", i)
		
			if f == nil {
				http.Error(w, fmt.Sprintf("idspec %s[%#v] not found", idspec, idspec), http.StatusInternalServerError)
				return
			}

			str += fmt.Sprintf("    %s\n", f.IdSpec())
			str += fmt.Sprintf("    %s\n", f.FullString())
			str += fmt.Sprintf("    airframe: %s\n", f.Airframe.String())
			str += fmt.Sprintf("    %s\n\n", f)
			str += fmt.Sprintf("    index tags: %v\n", f.IndexTagList())
			str += fmt.Sprintf("    /batch/flights/flight?flightkey=%s&job=retag\n", f.GetDatastoreKey())

			t := f.AnyTrack()
			str += fmt.Sprintf("\n---- Anytrack: %s\n", t)

			/*
			gr := geo.LatlongBoxRestrictor{LatlongBox: sfo.KFixes["ZORSA"].Box(1,1) }
			satisfies,intersection,deb := f.SatisfiesGeoRestriction(gr, []string{"ADSB","MLAT"})
			str += fmt.Sprintf("\n\n---- ZORSA@1KM intersection result:-\n * %v\n * %s\n----\n%s",
				satisfies, intersection, deb)*/
			
			/* pos := sfo.KFixes["BRIXX"]
		gr := geo.LatlongBoxRestrictor{LatlongBox: pos.Box(1,1) }
		isects,debug := t.AllIntersectsGeoRestriction(gr)
		str += fmt.Sprintf("---- Intersections\n")
		for _,isect := range isects { str += fmt.Sprintf("  -- %s\n", isect) }
		str += fmt.Sprintf("\n%s", debug) */
			
			for k,v := range f.Tracks {
				str += fmt.Sprintf("  -- [%-7.7s] %s\n", k, v)
				if r.FormValue("v") != "" {
					for n,tp := range *v {
						str += fmt.Sprintf("    - [%3d] %s\n", n, tp)
					}
				}
			}
			str += "\n"
			str += fmt.Sprintf("--- DebugLog:-\n%s\n", f.DebugLog)
		}
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\n\n%s", str)))
}

// }}}

// {{{ AddFragEvent{}

type AddFragEvent struct {
	TrackFragment fdb.TrackFragment
	Time time.Time
	I int `json:"-"`
	N int `json:"-"`  // Added N trackpoints starting at index I
	J int `json:"-"` // result number
}

func (ev AddFragEvent)String() string {
	// Delay from when message was received, to when frag was written to datastore
	delay := ev.Time.Sub(ev.TrackFragment.Track[0].TimestampUTC).Seconds()
	return fmt.Sprintf("{{%2d}} % 4d + %2d : %s (total delay %6.0fs) %s",
		ev.J, ev.I, ev.N,
		ev.TrackFragment.Track[0].TimestampUTC.Format("15:04:05"),
		delay, ev.TrackFragment)
}


type ByTimeOfDBWrite []AddFragEvent

func (a ByTimeOfDBWrite) Len() int           { return len(a) }
func (a ByTimeOfDBWrite) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTimeOfDBWrite) Less(i, j int) bool { return a[i].Time.Before(a[j].Time) }

// }}}
// {{{ AddNewEvent

func AddNewEvent(events *[]AddFragEvent, instance int, t fdb.Track, id,callsign string, start,length int, tm time.Time) {
	ev := AddFragEvent{
		TrackFragment: track2frag(t, id, callsign, start, length),
		Time:tm,
		J:instance,
		I:start,
		N:length,
	}

	*events = append(*events, ev)
}

func track2frag(t fdb.Track, id,callsign string, start,length int) fdb.TrackFragment {
	if len(t) == 0 || start+length > len(t) {
		return fdb.TrackFragment{}
	}
	
	frag := fdb.TrackFragment{
		IcaoId: adsb.IcaoId(id),
		Callsign: callsign,
		Track: t[start:(start+length)],
		DataSystem: fdb.DSADSB,
	}
	
	if t[0].DataSource != "ADSB" {
		frag.DataSystem = fdb.DSMLAT
	}

	return frag
}

// }}}
// {{{ debugFragsHandler

// This handler reconstructs the series of TrackFragments that generated the set of
// flights matching the idpsec (use a range idspec to get multiple instances of the same IcaoID).
// It parses the debug log to figure out which trackpoints got added when, so it's brittle.
func debugFragsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,_ := GetUIOptions(ctx)
	db := fgae.NewDB(ctx)
	str := ""

	idspecs,err := opt.IdSpecs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	events := []AddFragEvent{}		
	nPts := 0

	for _,idspec := range idspecs {
		str += fmt.Sprintf("*** %s [%v]\n", idspec, idspec)

		results,err := db.LookupAll(db.NewQuery().ByIdSpec(idspec))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		for i,result := range results {
			str += fmt.Sprintf("  * [%02d] %s\n", i, result)
			nPts += len(result.AnyTrack())
		}
		str += fmt.Sprintf("*** Total pts: %d\n\n\n", nPts)
		
		for instance,f := range results {			
			var tStart time.Time
			var track fdb.Track
			firstHandled := false

			// What to write the fragments as
			newIcao := r.FormValue("as")
			if newIcao == "" { newIcao = f.IcaoId }
			
			for _,line := range strings.Split(f.DebugLog, "\n") {
				exp := "-- AddFrag.*\\]([AM]) (\\d{4}-.* UTC): "+
					"(was not plausible|new IcaoID|extending \\(adding (\\d+) to (\\d+))"
				reg := regexp.MustCompile(exp).FindStringSubmatch(line)
				if reg != nil {
					t,_ := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", reg[2])
					length,_ := strconv.Atoi(reg[4])
					index,_ := strconv.Atoi(reg[5])

					if length == 0 { // AKA undef; must be the initial fragment; we don't know how long it is
						tStart = t
						switch (fdb.DataSystem(reg[1])) {
						case fdb.DSADSB: track = *(f.Tracks["ADSB"])
						default: track = *(f.Tracks["MLAT"])
						}

					} else {
						if !firstHandled {
							// Now we know how long the first frag was, add it in
							if 0+index > len(track) {
								http.Error(w, fmt.Sprintf("0+%d > %d !", index, len(track)),
									http.StatusInternalServerError)
								return
							}
							AddNewEvent(&events, instance, track, newIcao, f.Callsign, 0, index, tStart)
							firstHandled = true
						}
						if index+length > len(track) {
							http.Error(w, fmt.Sprintf("%d+%d > %d !", index, length, len(track)),
								http.StatusInternalServerError)
							return
						}
						AddNewEvent(&events, instance, track, newIcao, f.Callsign, index, length, t)
					}
				}
			}

			if !firstHandled {
				// We only ever had one frag - so its length is that of the whole track
				AddNewEvent(&events, instance, track, newIcao, f.Callsign, 0, len(track), tStart)
			}
		}
	}

	//fgae.Debug = true
	
	sort.Sort(ByTimeOfDBWrite(events)) // Restore the original insertion order

	if false && len(events) == 865 {
		// The original list of 864 frags is too long to use with test datastore. Prune out these
		// uninteresting serial ones: [776,864] [300,758] [143,260]
		events = events[:777]
		events = append(events[:300], events[759:]...)
		events = append(events[:143], events[261:]...)
	}
	
	for i,ev := range events {
		relativeT := ev.Time.Sub(events[0].Time).Seconds() // Time from when first frag was written

		if r.FormValue("as") != "" {
			// We've been asked to reinsert these frags, in their orig order, with a new IcaoID
			time.Sleep(1000 * time.Millisecond) // the test datastore gets confused if we go any quicker
			err := db.AddTrackFragment(&ev.TrackFragment)
			str += fmt.Sprintf("%08.3fs %s added [[%v]]\n", relativeT, ev, err)
		} else if r.FormValue("json") == "" {
			str += fmt.Sprintf("[% 5d] %08.3fs %s\n", i, relativeT, ev)
		}
	}
	
	if r.FormValue("json") != "" {
		frags := []fdb.TrackFragment{}
		for _,ev := range events {
			frags = append(frags, ev.TrackFragment)
		}
		json,_ := json.MarshalIndent(frags, "", "  ")
		str = "OK\n--\n"+string(json)
	}
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\n\n%s", str)))
}

// }}}

// {{{ AirspaceDelayHandler

// Pulls a skypi airspace, resyncs flight positions to some point in past (or future),
// and displays them both with a line between them.
//	&sync=-20s

func AirspaceDelayHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client := urlfetch.Client(ctx)
	pos := geo.Latlong{37.060312,-121.990814}

	syncedAge := widget.FormValueDuration(r, "sync")
	if syncedAge == 0 { syncedAge = 2 * time.Minute }
	
	as,err := fr24.FetchAirspace(client, pos.Box(100,100))
	if err != nil {
		http.Error(w, fmt.Sprintf("FetchAirspace: %v", err), http.StatusInternalServerError)
		return
	}
	ms := NewMapShapes()
	for _,ad := range as.Aircraft {
		tp := fdb.TrackpointFromADSB(ad.Msg)
		age := time.Since(tp.TimestampUTC)
		rewindDuration := age - syncedAge
		newTp := tp.RepositionByTime(rewindDuration)

		ms.AddIcon(MapIcon{TP:&tp,    Color:"#404040", Text:ad.Msg.Callsign,     ZIndex:1000})
		ms.AddIcon(MapIcon{TP:&newTp, Color:"#c04040", Text:ad.Msg.Callsign+"'", ZIndex:2000})
		ms.AddLine(MapLine{Start:tp.Latlong, End:newTp.Latlong, Color:"#000000"})
	}
	
	var params = map[string]interface{}{
		"Legend": "Debug Thing",
		"Center": sfo.KFixes["YADUT"],
		"Zoom": 9,
	}
	
	MapHandlerWithShapesParams(ctx, w, r, ms, params);
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

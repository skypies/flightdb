package main

import(
	"fmt"
	"net/http"
	
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	"golang.org/x/net/context"

	"github.com/skypies/adsb"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/widget"
	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fgae"
	"github.com/skypies/flightdb2/fr24"
	"github.com/skypies/flightdb2/ref"
)

func init() {
	http.HandleFunc("/fdb/tracks", trackHandler)
}

// {{{ maybeAddFr24Track

func MaybeAddFr24Track(c context.Context, f *fdb.Flight) string {
	fr,_ := fr24.NewFr24(urlfetch.Client(c))
	fr24Id,debug,err := fr.GetFr24Id(f)
	str := fmt.Sprintf("** fr24 ID lookup: %s, %v\n* debug:-\n%s***\n", fr24Id, err, debug)

	if fr24Id == "" { return str }
	
	var tF *fdb.Track
	if fr24Flight,err := fr.LookupPlaybackTrack(fr24Id); err != nil {
		str += fmt.Sprintf("* fr24 tracklookup: err: %v\n", err)
	} else {
		// TODO: sanity check this found flight is anything sensible at all
		str += fmt.Sprintf("* fr24 tracklookup found: %s\n", fr24Flight.IdentityString())
		tF = fr24Flight.Tracks["fr24"]
	}

	str += fmt.Sprintf("* [r2] %-6.6s : %s\n", "fr24", tF)

	for name,t := range f.Tracks {
		str += fmt.Sprintf("* [r1] %-6.6s : %s\n", name, t)
		overlaps, conf, debug := t.OverlapsWith(*tF)
		str += fmt.Sprintf("* --> %v, %f\n%s\n***\n", overlaps, conf, debug)
	}

	f.Tracks["fr24"] = tF
	
	return str
}

// }}}

// {{{ trackHandler

// ?icaoid=A12123 t=14123123123 max=3 colorby=rcvr fr24=1 debug=1 boxes=1
//  boxes=fr24  (see boxes for just that track)
//  track=fr24  (see dots for just that track)

func trackHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	db := fgae.FlightDB{C:c}

	q := db.NewQuery().ByIcaoId(adsb.IcaoId(r.FormValue("icaoid")))
	if r.FormValue("t") != "" {
		q = q.ByTime(widget.FormValueEpochTime(r,"t"))
	}

	flights,err := db.LookupAll(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if len(flights) == 0 {
		http.Error(w, "No matching flights found", http.StatusInternalServerError)
		return
	}
	// If there are lots of matches, trim to max
	maxTracks := widget.FormValueInt64(r, "max")
	if maxTracks > 0 && int64(len(flights)) > maxTracks {
		flights = append(flights[:maxTracks])
	}
	
	allText := ""
	for i,_ := range flights {
		allText += fmt.Sprintf("*** %s (%d) %s %s\n", flights[i].IdentityString(),
			len(flights[i].AnyTrack()),
			flights[i].AnyTrack().Start(),
			flights[i].GetLastUpdate())
	}
	// This whole Airframe cache thing should be automatic, and upstream from here.
	airframes := ref.NewAirframeCache(c)
	for i,_ := range flights {
		if af := airframes.Get(flights[i].IcaoId); af != nil {
			flights[i].Airframe = *af
		}
	}

	points := []MapPoint{}
	lines  := []MapLine{}
	
	coloring := ByCandyStripe
	switch r.FormValue("colorby") {
	case "src":
		coloring = ByDataSource
	case "rcvr":
		coloring = ByADSBReceiver
	}

	// Live fetch, and overlay, a track from fr24.
	if r.FormValue("fr24") != "" {
		coloring = ByDataSource
		allText += MaybeAddFr24Track(c, flights[0])
	}
		
	if r.FormValue("debug") != "" {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(fmt.Sprintf("OK\n\n%s", allText)))
		return
	}
	if len(flights) > 1 {
		// For each flight, translate a track into JS points, add to a JSPointSet
		color := "blue"
		for _,f := range flights {
			text := allText + fmt.Sprintf("* %s", f.IdentString())
			points = append(points, TrackToMapPoints(f.Tracks["ADSB"], color, text, coloring)...)
			if color == "blue" { color = "yellow" } else { color = "blue" }
		}

	} else {
		f := flights[0]
		// Pick most recent instance, and colorize all visible tracks.
		for _,trackType := range []string{"ADSB", "fr24", "FA:TA", "FA:TZ"} {
			if len(r.FormValue("track")) > 1 && r.FormValue("track") != trackType { continue }
			if _,exists := f.Tracks[trackType]; !exists { continue }
			points = append(points, TrackToMapPoints(f.Tracks[trackType], "", allText, coloring)...)
		}

		// &boxes=1
		if r.FormValue("boxes") != "" {
			for name,color := range map[string]string{
				"ADSB":"#888811","fr24":"#11aa11","FA:TA":"#1111aa","FA:TZ":"#1111aa",
			} {
				if len(r.FormValue("boxes")) > 1 && r.FormValue("boxes") != name { continue }
				if t,exists := f.Tracks[name]; exists==true {
					for _,box := range t.AsContiguousBoxes() {
						lines = append(lines, LatlongTimeBoxToMapLines(box, color)...)
					}
				}
			}
		}
	}
	
	legend := flights[0].IdentString()
	if len(flights)>1 { legend += fmt.Sprintf(" (%d instances)", len(flights)) }
	
	var params = map[string]interface{}{
		"Legend": legend,
		"Points": MapPointsToJSVar(points),
		"Lines": MapLinesToJSVar(lines),
		"MapsAPIKey": "",//kGoogleMapsAPIKey,
		"Center": sfo.KFixes["EPICK"], //sfo.KLatlongSFO,
		"Zoom": 8,
	}

	if err := templates.ExecuteTemplate(w, "fdb-tracks", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

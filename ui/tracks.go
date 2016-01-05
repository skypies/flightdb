package main

import(
	"fmt"
	"net/http"
	
	"google.golang.org/appengine"

	"github.com/skypies/adsb"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/widget"
	"github.com/skypies/flightdb2/fgae"
)

func init() {
	http.HandleFunc("/fdb/tracks", trackHandler)
}

// {{{ trackHandler

// ?icaoid=A12123[&t=14123123123]

func trackHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	db := fgae.FlightDB{C:c}

	q := db.NewQuery().ByIcaoId(adsb.IcaoId(r.FormValue("icaoid")))

	if r.FormValue("t") != "" {
		t := widget.FormValueEpochTime(r,"t")
		_=t
	}
	
	flights,err := db.LookupAll(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// For each flight, translate a track into JS points, add to a JSPointSet
	points := []MapPoint{}
	color := "blue"
	for _,f := range flights {
		text := fmt.Sprintf("* DKey: %s", f.GetDatastoreKey())
		points = append(points, TrackToMapPoints(f.Tracks["ADSB"], color, text)...)
		if color == "blue" { color = "yellow" } else { color = "blue" }
	}
		
	var params = map[string]interface{}{
		"Legend": fmt.Sprintf("%s (%d instances)", flights[0].IdentString(), len(flights)),
		"Points": MapPointsToJSVar(points),
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

package ui

import(
	"encoding/json"
	"html/template"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/net/context"

	"github.com/skypies/util/date"
	"github.com/skypies/util/widget"
	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"

	"github.com/skypies/flightdb/fgae"
)

// {{{ buildLegend

func legendUrl(t time.Time, offset int64, val string) string {
	epoch := t.Unix() + offset
	return fmt.Sprintf("<a href=\"/fdb/historical?epoch=%d\">%s</a>", epoch, val)
}

func buildLegend(t time.Time) string {
	legend := date.InPdt(t).Format("15:04:05 MST (2006/01/02)")

	legend += " ["+
		legendUrl(t,-3600,"-1h")+", "+
		legendUrl(t,-1200,"-20m")+", "+
		legendUrl(t, -600,"-10m")+", "+
		legendUrl(t, -300,"-5m")+", "+
		legendUrl(t,  -60,"-1m")+", "+
		legendUrl(t,  -30,"-30s")+"; "+
		legendUrl(t,   30,"+30s")+"; "+
		legendUrl(t,   60,"+1m")+", "+
		legendUrl(t,  300,"+5m")+", "+
		legendUrl(t,  600,"+10m")+", "+
		legendUrl(t, 1200,"+20m")+", "+
		legendUrl(t, 3600,"+1h")+
		"]"
	return legend
}

// }}}

// {{{ HistoricalHandler

// /fdb/historical?
//  epoch=141041412424214     or    date=2016/02/28&time=16:40:20
//  pos_lat=36.0&pos_long=-122.0
//  resultformat=json  (or list or map ?)

func HistoricalHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	templates,_ := GetTemplates(ctx)

	if r.FormValue("date") == "" && r.FormValue("epoch") == "" {
		var params = map[string]interface{}{
			"TwoHoursAgo": date.NowInPdt().Add(-10 * time.Minute),
		}
		if err := templates.ExecuteTemplate(w, "fdb-historical-form", params); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	db := fgae.NewDBFromReq(r)

	var t time.Time
	if r.FormValue("epoch") != "" {
		t = widget.FormValueEpochTime(r, "epoch")
	} else {
		var err error
		t,err = date.ParseInPdt("2006/01/02 15:04:05", r.FormValue("date")+" "+r.FormValue("time"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	refPoint := geo.FormValueLatlong(r, "pos")
	
	as,err := db.LookupHistoricalAirspace(t.UTC(), refPoint, 1000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	aircraftJSON,err := json.MarshalIndent(as, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.FormValue("resultformat") == "json" {
		w.Header().Set("Content-Type", "application/json")
		w.Write(aircraftJSON)
		return
	}
		
	var params = map[string]interface{}{
		"Legend": buildLegend(t),
		"SearchTimeUTC": t.UTC(),
		"SearchTime": date.InPdt(t),
		//"AirspaceJS": as.ToJSVar(r.URL.Host, t),
		"AircraftJSON": template.JS(aircraftJSON),
		"MapsAPIKey": "",
		"Center": sfo.KFixes["YADUT"],
		"Waypoints": WaypointMapVar(sfo.KFixes),
		"Zoom": 9,
	}

	if err := templates.ExecuteTemplate(w, "fdb-historical-map", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

package ui

import(
	"fmt"
	"net/http"
	"golang.org/x/net/context"
)

func init() {
}

// ?idspec=XX,YY,...  (or ?idspec=XX&idspec=YYY&...)
//  &viewtype={vector,descent,sideview,track}
//  &sample=N        (sample the track every N seconds)

//  &alt=30000       (max altitude for graph)
//  &length=80       (max distance from origin; in nautical miles)
//  &dist=from       (for distance axis, use dist from airport; by default, uses dist along path)
//  &colorby=delta   (delta groundspeed, instead of groundspeed)

func VisualizeHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.FormValue("debug") != "" {
		str := "OK\n"
		for k, v := range r.Form {
			str += fmt.Sprintf(" %-20.20s: '%s'\n", k, v)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(str))
	}
	
	switch r.FormValue("viewtype") {
	case "vector":   TracksetHandler(ctx,w,r)
	//case "descent":  escentHandler(ctx,w,r)
	case "sideview": SideviewHandler(ctx,w,r)
	case "track":    TrackHandler(ctx,w,r)
	default:         http.Error(w, "Specify viewtype={vector|sideview|track}", http.StatusBadRequest)
	}		
}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

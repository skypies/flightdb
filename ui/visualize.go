package ui

import(
	"fmt"
	"net/http"
)

func init() {
	http.HandleFunc("/fdb/visualize", visualizeHandler)
}

// ?idspec=XX,YY,...  (or ?idspec=XX&idspec=YYY&...)
//  &sample=N        (sample the track every N seconds)
//  &alt=30000       (max altitude for graph)
//  &length=80       (max distance from origin; in nautical miles)
//  &dist=from       (for distance axis, use dist from airport; by default, uses dist along path)
//  &colorby=delta   (delta groundspeed, instead of groundspeed)

func visualizeHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	if r.FormValue("debug") != "" {
		str := "OK\n"
		for k, v := range r.Form {
			str += fmt.Sprintf(" %-20.20s: '%s'\n", k, v)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(str))
	}
	
	switch r.FormValue("viewtype") {
	case "vector":   tracksetHandler(w,r)
	case "descent":  descentHandler(w,r)
	case "track":    trackHandler(w,r)
	default:         http.Error(w, "Specify viewtype={vector|descent|track}", http.StatusBadRequest)
	}		
}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

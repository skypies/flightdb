package main

import(
	"fmt"
	"net/http"
	
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	//"google.golang.org/appengine/log"

	"github.com/skypies/geo/sfo"
	"github.com/skypies/flightdb2/fr24"
	"github.com/skypies/flightdb2/ref"
)

func init() {
	http.HandleFunc("/fdb/fr24", fr24PollHandler)
}

// dev_appserver.py --clear_datastore=yes ./ui.yaml

func fr24PollHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	// db := fgae.FlightDB{C:c}
	fr,_ := fr24.NewFr24(urlfetch.Client(c))

	str := ""
	box := sfo.KLatlongSFO.Box(160,160)  // This is the box in which we look for new flights	

	resp,err := fr.LookupCurrentList(box)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	airframes := ref.NewAirframeCache(c)
	for _,fs := range resp {
		if fs.Registration == "" || airframes.Get(fs.IcaoId) != nil { continue }
		af := ref.Airframe{fs.IcaoId, fs.Registration, fs.EquipmentType}
		airframes.Set(&af)
		str += fmt.Sprintf(" ** New [%s] %s\n", fs.Registration, fs)
	}

	if err := airframes.Persist(c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.FormValue("list") != "" {
		str += fmt.Sprintf("\n%s\n", airframes)
	} else {
		str += fmt.Sprintf("\nhave %d airframes stored\n", len(airframes.Map))
	}
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\n\n%s", str)))
}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

package main

import(
	"fmt"
	"net/http"
	
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	//"google.golang.org/appengine/log"

	"github.com/skypies/geo/sfo"
	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fr24"
	"github.com/skypies/flightdb2/ref"
)

func init() {
	http.HandleFunc("/fdb/fr24", fr24PollHandler)
}

// dev_appserver.py --clear_datastore=yes ./ui.yaml

func listResult2Airframe(fs fdb.FlightSnapshot) ref.Airframe {
	af := ref.Airframe{
		Icao24:fs.IcaoId,
		Registration:fs.Registration,
		EquipmentType:fs.EquipmentType,
	}

	callsign := fdb.ParseCallsignString(fs.Callsign)
	if callsign.CallsignType == fdb.IcaoFlightNumber {
		af.CallsignPrefix = callsign.IcaoPrefix
	}
	
	return af
}

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
	n := 0
	for _,fs := range resp {
		if fs.Registration == "" { continue }
		newAf := listResult2Airframe(fs)
		oldAf := airframes.Get(fs.IcaoId)

		if oldAf == nil || oldAf.CallsignPrefix != newAf.CallsignPrefix {
			airframes.Set(&newAf)
			str += fmt.Sprintf("* [%7.7s]%s %s\n", fs.Registration, newAf, fs)
			n++
		}
	}

	if n>0 {
		if err := airframes.Persist(c); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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

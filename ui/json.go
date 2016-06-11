package ui

import(
	"encoding/json"
	"fmt"
	"net/http"

	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fgae"
	"github.com/skypies/flightdb2/ref"
)

func init() {
	http.HandleFunc("/fdb/json", jsonHandler)
	http.HandleFunc("/fdb/snarf", snarfHandler)
}

// {{{ LookupIdspec

func LookupIdspec(db fgae.FlightDB, idspec fdb.IdSpec) ([]*fdb.Flight, error) {
	flights := []*fdb.Flight{}
	
	if idspec.Duration == 0 {
		// This is a point-in-time idspec; we want the single, most recent match only
		if result,err := db.LookupMostRecent(db.NewQuery().ByIdSpec(idspec)); err != nil {
			return flights, err
		} else {
			flights = append(flights, result)
		}

	} else {
		// This is a range idspec; we want everything that matches.
		if results,err := db.LookupAll(db.NewQuery().ByIdSpec(idspec)); err != nil {
			return flights, err
		} else {
			flights = append(flights, results...)
		}
	}
	return flights, nil
}

// }}}

// {{{ jsonHandler

func jsonHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	// This whole Airframe cache thing should be automatic, and upstream from here.
	airframes := ref.NewAirframeCache(c)

	idspecs,err := FormValueIdSpecs(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	db := fgae.FlightDB{C:c}
	flights := []*fdb.Flight{}
	for _,idspec := range idspecs {
		if results,err := LookupIdspec(db, idspec); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if len(results) == 0 {
			http.NotFound(w,r)
			//http.Error(w, fmt.Sprintf("idspec %s not found", idspec), http.StatusNotFound)
			return
		} else {
			for _,f := range results {
				if f == nil { continue }  // Bad input data ??
				if af := airframes.Get(f.IcaoId); af != nil {
					f.Airframe = *af
				}
				if r.FormValue("notracks") != "" {
					f.PruneTrackContents()
				}
				flights = append(flights, f)
			}
		}
	}

	jsonBytes,err := json.MarshalIndent(flights, "", " ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonBytes)
}

// }}}
// {{{ snarfHandler

func snarfHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	client := urlfetch.Client(c)
	db := fgae.FlightDB{C:c}

	//url := "http://stop.jetnoise.net/fdb/json2?idspec=" + r.FormValue("idspec")
	url := "http://fdb.serfr1.org/fdb/json?idspec=" + r.FormValue("idspec")

	str := fmt.Sprintf("Snarfer!\n--\n%s\n--\n", url)
	
	flights := []fdb.Flight{}
	if resp,err := client.Get(url); err != nil {
		http.Error(w, "XX: "+err.Error(), http.StatusInternalServerError)
		return
	} else {
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf ("Bad status: %v", resp.Status)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if err := json.NewDecoder(resp.Body).Decode(&flights); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	str += "Snarfed:-\n"
	for _,f := range flights {
		str += fmt.Sprintf(" * %s\n", f)
	}
	str += "--\n"

	for _,f := range flights {
		if err := db.PersistFlight(&f); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	str += "all persisted OK!\n"
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(str))
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

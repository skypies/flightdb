package ui

import(
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/appengine/urlfetch"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/fgae"
	"github.com/skypies/flightdb/ref"
)

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

// {{{ JsonHandler

// /fdb/json?idspec=...  - dumps an entire flight object out as JSON.

func JsonHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,_ := GetUIOptions(ctx)
	db := fgae.NewDB(ctx)

	// This whole Airframe cache thing should be automatic, and upstream from here.
	airframes := ref.NewAirframeCache(ctx)

	idspecs,err := opt.IdSpecs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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
// {{{ SnarfHandler

// /fdb/snarf?idspec=...  - pull the idspecs from prod, insert into local DB. For debugging.

func SnarfHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,_ := GetUIOptions(ctx)
	client := urlfetch.Client(ctx)
	db := fgae.NewDB(ctx)

	str := "Snarfer!\n--\n\n"

	idspecs,err := opt.IdSpecs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	allFlights := []fdb.Flight{}
	for _,idspec := range idspecs {
		theseFlights := []fdb.Flight{}
		url := fmt.Sprintf("http://fdb.serfr1.org/fdb/json?idspec=%s", idspec)
		str += fmt.Sprintf("-- snarfing: %s\n", url)
	
		if resp,err := client.Get(url); err != nil {
			http.Error(w, "XX: "+err.Error(), http.StatusInternalServerError)
			return
		} else {
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				err = fmt.Errorf ("Bad status: %v", resp.Status)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			} else if err := json.NewDecoder(resp.Body).Decode(&theseFlights); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		allFlights = append(allFlights, theseFlights...)

		str += "-- Found:-\n"
		for _,f := range theseFlights {
			str += fmt.Sprintf(" * %s\n", f)
		}
		str += "--\n"
	}
	
	for _,f := range allFlights {
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

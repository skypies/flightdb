package ui

import(
	"encoding/json"
	"fmt"
	"net/http"
	
	"google.golang.org/appengine"
	//"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/user"

	"github.com/skypies/flightdb2/fgae"
)

func init() {
	http.HandleFunc("/fdb/debug", debugHandler) // should rename at some point ...
	http.HandleFunc("/fdb/debug/user", debugUserHandler)
}

// {{{ debugUserHandler

func debugUserHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	user := user.Current(c)
	json,_ := json.MarshalIndent(user, "", "  ")

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("OK\n--\n"+string(json)))
}

// }}}
// {{{ debugHandler

func debugHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	str := ""
	
	idspecs,err := FormValueIdSpecs(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//str += fmt.Sprintf("** Idspecs:-\n%#v\n\n", idspecs)

	db := fgae.FlightDB{C:c}	
	for _,idspec := range idspecs {
		str += fmt.Sprintf("*** %s [%v]\n", idspec, idspec)
		f,err := db.LookupMostRecent(db.NewQuery().ByIdSpec(idspec))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if f == nil {
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
		str += fmt.Sprintf("---- Anytrack: %s\n", t)

		for k,v := range f.Tracks {
			str += fmt.Sprintf("  -- [%-7.7s] %s\n", k, v)
			if r.FormValue("v") != "" {
				for n,tp := range *v {
					str += fmt.Sprintf("    - [%3d] %s\n", n, tp)
				}
			}
		}

		str += fmt.Sprintf("\n--- DebugLog:-\n%s\n", f.DebugLog)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\n\n%s", str)))
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

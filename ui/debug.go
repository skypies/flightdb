package ui

import(
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/appengine/user"

	"github.com/skypies/flightdb2/fgae"
)

func init() {
	http.HandleFunc("/fdb/debug", UIOptionsHandler(debugHandler)) // should rename at some point ...
	http.HandleFunc("/fdb/debug/user", UIOptionsHandler(debugUserHandler))
}

// {{{ debugUserHandler

func debugUserHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	user := user.Current(ctx)
	json,_ := json.MarshalIndent(user, "", "  ")

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("OK\n--\n"+string(json)))
}

// }}}
// {{{ debugHandler

func debugHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,_ := GetUIOptions(ctx)
	db := fgae.FlightDB{C:ctx}
	str := ""

	//str += fmt.Sprintf("** Idspecs:-\n%#v\n\n", opt.IdSpecs())

	idspecs,err := opt.IdSpecs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _,idspec := range idspecs {
		str += fmt.Sprintf("*** %s [%v]\n", idspec, idspec)

		results,err := db.LookupAll(db.NewQuery().ByIdSpec(idspec))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		for i,result := range results {
			s,e := result.Times()
			str += fmt.Sprintf("  * [%02d] %s,%s  %s\n", i, s,e, result.IdentityString())
		}
		str += "\n\n\n\n"
		
		for i,f := range results {
			str += fmt.Sprintf("----------{result %02d }-----------\n\n", i)
		
			if f == nil {
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

			/* pos := sfo.KFixes["BRIXX"]
		gr := geo.LatlongBoxRestrictor{LatlongBox: pos.Box(1,1) }
		isects,debug := t.AllIntersectsGeoRestriction(gr)
		str += fmt.Sprintf("---- Intersections\n")
		for _,isect := range isects { str += fmt.Sprintf("  -- %s\n", isect) }
		str += fmt.Sprintf("\n%s", debug) */
			
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

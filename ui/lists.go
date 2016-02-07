package ui

import(
	"net/http"
	
	"google.golang.org/appengine"
	//"google.golang.org/appengine/log"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fgae"
	//"github.com/skypies/flightdb2/ref"
)

func init() {
	http.HandleFunc("/fdb/recent2", listHandler)
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	db := fgae.FlightDB{C:c}

	tags := []string{}
	flights := []*fdb.Flight{}

	//airframes := ref.NewAirframeCache(c)
	
	iter := db.NewIterator(db.QueryForRecent(tags, 200))
	for iter.Iterate() {
		if iter.Err() != nil {
			http.Error(w, iter.Err().Error(), http.StatusInternalServerError)
			return
		}
		f := iter.Flight()
		f.PruneTrackContents() // Save on RAM
		flights = append(flights, f)
	}

	var params = map[string]interface{}{
		"Flights": flights,
	}
	if err := templates.ExecuteTemplate(w, "fdb-recentlist", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

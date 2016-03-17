package ui

import(
	"net/http"
	
	"google.golang.org/appengine"
	//"google.golang.org/appengine/log"

	"github.com/skypies/util/widget"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fgae"
	//"github.com/skypies/flightdb2/ref"
)

func init() {
	http.HandleFunc("/fdb/list", listHandler)
}

// icaoid=A12345 - lookup recent flights on that airframe
func listHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	db := fgae.FlightDB{C:c}

	tags := widget.FormValueCommaSepStrings(r, "tags")
	flights := []*fdb.Flight{}
	
	//airframes := ref.NewAirframeCache(c)
	query := db.QueryForRecent(tags, 200)
	if r.FormValue("icaoid") != "" {
		query = db.QueryForRecentIcaoId(r.FormValue("icaoid"), 200)
	}
	
	iter := db.NewIterator(query)
	for iter.Iterate() {
		if iter.Err() != nil { break }
		f := iter.Flight()
		f.PruneTrackContents() // Save on RAM
		flights = append(flights, f)
	}
	if iter.Err() != nil {
		http.Error(w, iter.Err().Error(), http.StatusInternalServerError)
		return
	}
	
	var params = map[string]interface{}{
		"Tags": tags,
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

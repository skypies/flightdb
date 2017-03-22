package ui

import(
	"net/http"
	
	"google.golang.org/appengine"

	"github.com/skypies/util/widget"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/db"
)

func init() {
	http.HandleFunc("/fdb/list", listHandler)
}

// icaoid=A12345 - lookup recent flights on that airframe
func listHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	p := db.AppengineDSProvider{}

	tags := widget.FormValueCommaSepStrings(r, "tags")
	flights := []*fdb.Flight{}
	
	//airframes := ref.NewAirframeCache(c)
	query := db.QueryForRecent(tags, 200)
	if r.FormValue("icaoid") != "" {
		query = db.QueryForRecentIcaoId(r.FormValue("icaoid"), 200)
	}
	
	it := db.NewFlightIterator(ctx, p, query)
	for it.Iterate(ctx) {
		f := it.Flight()
		f.PruneTrackContents() // Save on RAM
		flights = append(flights, f)
	}
	if it.Err() != nil {
		http.Error(w, it.Err().Error(), http.StatusInternalServerError)
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

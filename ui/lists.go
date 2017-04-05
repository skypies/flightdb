package ui

import(
	"net/http"

	"golang.org/x/net/context"

	"github.com/skypies/util/widget"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/fgae"
)

// icaoid=A12345 - lookup recent flights on that airframe
func ListHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	templates,_ := GetTemplates(ctx)
	db := fgae.NewDB(ctx)

	tags := widget.FormValueCommaSepStrings(r, "tags")
	flights := []*fdb.Flight{}
	
	//airframes := ref.NewAirframeCache(c)
	query := fgae.QueryForRecent(tags, 200)
	if r.FormValue("icaoid") != "" {
		query = fgae.QueryForRecentIcaoId(r.FormValue("icaoid"), 200)
	}
	
	it := db.NewIterator(query)
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

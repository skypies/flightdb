package ui

import(
	"fmt"
	"net/http"

	"google.golang.org/appengine"
	
	"github.com/skypies/util/date"

	//fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fgae"
	"github.com/skypies/flightdb2/report"
)

func init() {
	http.HandleFunc("/fdb/report", reportHandler)
}

func reportHandler(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("rep") == "" {
		var params = map[string]interface{}{
			"Yesterday": date.NowInPdt().AddDate(0,0,-1),
			"Reports": report.ListReports(),
			"FormUrl": "/fdb/report",
		}
		if err := templates.ExecuteTemplate(w, "report3-form", params); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	
	c := appengine.NewContext(r)
	db := fgae.FlightDB{C:c}

	rep,err := report.SetupReport(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//airframes := ref.NewAirframeCache(c)
	//metar ? 
	
	tags := []string{}
	idspecs := []string{}
	iter := db.NewIterator(db.QueryForRecent(tags, 20))
	for iter.Iterate() {
		if iter.Err() != nil {
			http.Error(w, iter.Err().Error(), http.StatusInternalServerError)
			return
		}
		f := iter.Flight()

		if included,err := rep.Process(f); err != nil {
			http.Error(w, iter.Err().Error(), http.StatusInternalServerError)
			return
		} else if included {
			idspecs = append(idspecs, f.IdSpec())
		}
	}

	if r.FormValue("debug") != "" {
		str := ""
		for _,r := range rep.MetadataTable() {
			str += fmt.Sprintf("-> %v <-\n", r)
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(fmt.Sprintf("OK\n\n%v\n%s\n", idspecs, str)))
		return
	}

	var params = map[string]interface{}{
		"R": rep,
		"Metadata": rep.MetadataTable(),
	}
	if err := templates.ExecuteTemplate(w, "report3-results", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}	
}

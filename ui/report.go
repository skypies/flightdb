package ui

import(
	"fmt"
	"html/template"
	"net/http"
	"strings"
	
	"google.golang.org/appengine"
	
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/date"
	"github.com/skypies/flightdb2/fgae"
	"github.com/skypies/flightdb2/report"
)

func init() {
	http.HandleFunc("/fdb/report", reportHandler)
}

// {{{ ButtonPOST

func ButtonPOST(anchor, action string, idspecs []string) string {
	// Would be nice to view the complement - approaches of flights that did not match
	posty := fmt.Sprintf("<form action=\"%s\" method=\"post\" target=\"_blank\">", action)
	posty += fmt.Sprintf("<button type=\"submit\" name=\"idspec\" value=\"%s\" "+
		"class=\"btn-link\">%s</button>", strings.Join(idspecs,","), anchor)
	posty += "</form>\n"
	return posty
}

// }}}

func reportHandler(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("rep") == "" {
		var params = map[string]interface{}{
			"Yesterday": date.NowInPdt().AddDate(0,0,-1),
			"Reports": report.ListReports(),
			"FormUrl": "/report",
			"Waypoints": sfo.ListWaypoints(),
			"Title": "Reports (DB v2)",
		}
		if err := templates.ExecuteTemplate(w, "report3-form", params); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	
	c := appengine.NewContext(r)
	db := fgae.FlightDB{C:c}
	//airframes := ref.NewAirframeCache(c)

	rep,err := report.SetupReport(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.FormValue("debug") != "" {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(fmt.Sprintf("OK\n--\n%s\n", rep.Options)))
		return
	}
	
	tags := []string{}
	idspecs := []string{}
	idspecComplement := []string{}
	iter := db.NewIterator(db.QueryForRecent(tags, 20))
	for iter.Iterate() {
		if iter.Err() != nil {
			http.Error(w, iter.Err().Error(), http.StatusInternalServerError)
			return
		}
		f := iter.Flight()

		if included,err := rep.Process(f); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if included {
			idspecs = append(idspecs, f.IdSpec())
		} else {
			idspecComplement = append(idspecComplement, f.IdSpec())
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

	postButtons := ButtonPOST("Matches as a VectorMap", fmt.Sprintf("/fdb/trackset?%s",
		rep.ToCGIArgs()), idspecs)
	postButtons += ButtonPOST("Non-matches as a VectorMap", fmt.Sprintf("/fdb/trackset?%s",
		rep.ToCGIArgs()), idspecComplement)
	if rep.Name == "sfoclassb" {
		postButtons += ButtonPOST("Matches as ClassBApproaches", fmt.Sprintf("/fdb/approach?%s",
			rep.ToCGIArgs()), idspecs)
	}

	var params = map[string]interface{}{
		"R": rep,
		"Metadata": rep.MetadataTable(),
		"PostButtons": template.HTML(postButtons),
		"IdSpecs": template.HTML(strings.Join(idspecs,",")),
		"DebugLog": rep.DebugLog,
		"Title": "Reports (DB v2)",
	}
	if err := templates.ExecuteTemplate(w, "report3-results", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}	
}

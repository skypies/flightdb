package main

import(
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"
	
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/date"
	"github.com/skypies/flightdb2/fgae"
	"github.com/skypies/flightdb2/report"
	_ "github.com/skypies/flightdb2/analysis" // populate the reports registry
)

func init() {
	http.HandleFunc("/report", reportHandler)
}

func ButtonPOST(anchor, action string, idspecs []string) string {
	// Would be nice to view the complement - approaches of flights that did not match
	posty := fmt.Sprintf("<form action=\"%s\" method=\"post\" target=\"_blank\">", action)
	posty += fmt.Sprintf("<button type=\"submit\" name=\"idspec\" value=\"%s\" "+
		"class=\"btn-link\">%s</button>", strings.Join(idspecs,","), anchor)
	posty += "</form>\n"
	return posty
}

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
	
	c,_ := context.WithTimeout(appengine.NewContext(r), 10 * time.Minute)

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
	
	idspecsAccepted := []string{}
	idspecsRejectByRestrict := []string{}
	idspecsRejectByReport := []string{}

	query := db.QueryForTimeRangeWaypoint(rep.Tags, rep.Options.HackWaypoints, rep.Start,rep.End)
	rep.Debug(fmt.Sprintf("Using DB Query:-\n%s\n%#v\n", query, query))
		
	iter := db.NewLongIterator(query)
	n := 0
	tStart := time.Now()
	for iter.Iterate() {
		if iter.Err() != nil { break }
		f := iter.Flight()
		n++
		
		outcome,err := rep.Process(f)
		if err != nil {
			errStr := fmt.Sprintf("Process err after %d (%s): %v", n, time.Since(tStart), err.Error())
			http.Error(w, errStr, http.StatusInternalServerError)
			return
		}

		switch outcome {
		case report.RejectedByGeoRestriction:
			idspecsRejectByRestrict = append(idspecsRejectByRestrict, f.IdSpecString())
		case report.RejectedByReport:
			idspecsRejectByReport = append(idspecsRejectByReport, f.IdSpecString())
		case report.Accepted:
			idspecsAccepted = append(idspecsAccepted, f.IdSpecString())
		}
	}
	if iter.Err() != nil {
		errStr := fmt.Sprintf("Iter err after %d (%s): %v", n, time.Since(tStart), iter.Err().Error())
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}

	rep.FinishSummary()
	
	if r.FormValue("debug") != "" {
		str := ""
		for _,r := range rep.MetadataTable() {
			str += fmt.Sprintf("-> %v <-\n", r)
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(fmt.Sprintf("OK\n\n%v\n%s\n", idspecsAccepted, str)))
		return
	}

	idspecsInReport := idspecsAccepted
	idspecsInReport = append(idspecsInReport, idspecsRejectByReport...)	
	
	postButtons := ButtonPOST(fmt.Sprintf("%d matches as a VectorMap", len(idspecsAccepted)),
		fmt.Sprintf("/fdb/trackset?%s", rep.ToCGIArgs()), idspecsAccepted)

	postButtons += ButtonPOST(fmt.Sprintf("%d restriction rejects as a VectorMap",
		len(idspecsRejectByRestrict)),
		fmt.Sprintf("/fdb/trackset?%s", rep.ToCGIArgs()), idspecsRejectByRestrict)
	postButtons += ButtonPOST(fmt.Sprintf("%d report rejects as a VectorMap",
		len(idspecsRejectByReport)),
		fmt.Sprintf("/fdb/trackset?%s", rep.ToCGIArgs()), idspecsRejectByReport)

	if rep.Name == "sfoclassb" {
		postButtons += ButtonPOST(fmt.Sprintf("%d matches as ClassBApproaches", len(idspecsAccepted)),
			fmt.Sprintf("/fdb/approach?%s", rep.ToCGIArgs()), idspecsAccepted)
		postButtons += ButtonPOST(fmt.Sprintf("%d report rejects as ClassBApproaches",
			len(idspecsRejectByReport)),
			fmt.Sprintf("/fdb/approach?%s", rep.ToCGIArgs()), idspecsRejectByReport)

		postButtons += ButtonPOST(fmt.Sprintf("%d accept/reject as ClassBApproaches",
			len(idspecsInReport)),
			fmt.Sprintf("/fdb/approach?%s", rep.ToCGIArgs()), idspecsInReport)
	}

	var params = map[string]interface{}{
		"R": rep,
		"Metadata": rep.MetadataTable(),
		"PostButtons": template.HTML(postButtons),
		"IdSpecs": template.HTML(strings.Join(idspecsAccepted,",")),
		"Title": "Reports (DB v2)",
	}
	if err := templates.ExecuteTemplate(w, "report3-results", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}	
}

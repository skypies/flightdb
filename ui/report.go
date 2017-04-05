package ui

import(
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"
	
	"golang.org/x/net/context"
	
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/date"
	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/fgae"
	"github.com/skypies/flightdb/report"
	_ "github.com/skypies/flightdb/analysis" // populate the reports registry
)

func init() {
}

func ButtonPOST(anchor, action string, idspecs []string) string {
	action += "&colorby=altitude"
	posty := fmt.Sprintf("<form action=\"%s\" method=\"post\" target=\"_blank\">", action)
	posty += fmt.Sprintf("<button type=\"submit\" name=\"idspec\" value=\"%s\" "+
		"class=\"btn-link\">%s</button>", strings.Join(idspecs,","), anchor)
	posty += "</form>\n"
	return posty
}

func maybeButtonPOST(idspecs []string, title string, url string) string {
	if len(idspecs) == 0 { return "" }
	return ButtonPOST(
		fmt.Sprintf("%d %s", len(idspecs), title),
		fmt.Sprintf("%s", url),
		idspecs)
}

func ReportHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,_ := GetUIOptions(ctx)
	templates,_ := GetTemplates(ctx)
	db := fgae.NewDB(ctx)

	if r.FormValue("rep") == "" {
		// Show blank form
		var params = map[string]interface{}{
			"Yesterday": date.NowInPdt().AddDate(0,0,-1),
			"Reports": report.ListReports(),
			"FormUrl": "/report",
			"UIOptions": opt,
			"Waypoints": sfo.ListWaypoints(),
			"Title": fmt.Sprintf("Reports"),
		}

		if r.FormValue("log") != "" {
			params["LogLevel"] = r.FormValue("log")
		}
		
		fdb.BlankGeoRestrictorIntoParams(params)
		
		if opt.UserEmail != "" {
			if rsets,err := db.LookupRestrictorSets(opt.UserEmail); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			} else {
				params["RestrictorSets"] = rsets
			}
		} 

		if err := templates.ExecuteTemplate(w, "report-form", params); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	
	//airframes := ref.NewAirframeCache(c)

	rep,err := report.SetupReport(ctx, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.FormValue("debug") != "" {
		jsonBytes,_ := json.MarshalIndent(rep.Options, "", "  ")
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(fmt.Sprintf("OK\n--\n%s\n", string(jsonBytes))))//rep.Options)))
		return
	}

	idspecsAccepted := []string{}
	idspecsRejectByRestrict := []string{}
	idspecsRejectByReport := []string{}

	query := fgae.QueryForTimeRangeWaypoint(rep.Tags, rep.Options.Waypoints, rep.Start,rep.End)
	it := db.NewIterator(query)
	n := 0
	tStart := time.Now()
	tBottomOfLoop := tStart
	for it.Iterate(ctx) {
		rep.Stats.RecordValue("flightfetch", (time.Since(tBottomOfLoop).Nanoseconds()/1000))

		f := it.Flight()
		n++
		
		outcome,err := rep.Process(f)
		if err != nil {
			errStr := fmt.Sprintf("Process err after %d (%s): %v", n, time.Since(tStart), err.Error())
			http.Error(w, errStr, http.StatusInternalServerError)
			return
		}

		tBottomOfLoop = time.Now()
	
		switch outcome {
		case report.RejectedByGeoRestriction:
			idspecsRejectByRestrict = append(idspecsRejectByRestrict, f.IdSpecString())
		case report.RejectedByReport:
			idspecsRejectByReport = append(idspecsRejectByReport, f.IdSpecString())
		case report.Accepted:
			idspecsAccepted = append(idspecsAccepted, f.IdSpecString())
		}
	}
	if it.Err() != nil {
		errStr := fmt.Sprintf("Iter err after %d (%s): %v", n, time.Since(tStart), it.Err())
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

	if rep.ResultsFormat == "csv" {
		rep.OutputAsCSV(w)
		return
	}

	idspecsInReport := idspecsAccepted
	idspecsInReport = append(idspecsInReport, idspecsRejectByReport...)	
	
	postButtons := ""

	url := fmt.Sprintf("/fdb/trackset?%s", rep.ToCGIArgs())
	postButtons += maybeButtonPOST(idspecsRejectByRestrict, "Restriction Rejects as VectorMap",url)
	postButtons += maybeButtonPOST(idspecsRejectByReport, "Report Rejects as VectorMap", url)

  url = fmt.Sprintf("/fdb/descent?%s", rep.ToCGIArgs())
	postButtons += maybeButtonPOST(idspecsRejectByRestrict, "Restriction Rejects as DescentGraph",url)
	postButtons += maybeButtonPOST(idspecsRejectByReport, "Report Rejects as DescentGraph", url)
	
	if rep.Name == "sfoclassb" {
		url = fmt.Sprintf("/fdb/approach?%s", rep.ToCGIArgs())
		postButtons += maybeButtonPOST(idspecsAccepted, "Matches as ClassB",url)
		postButtons += maybeButtonPOST(idspecsRejectByRestrict, "Restriction Rejects as ClassB",url)
		postButtons += maybeButtonPOST(idspecsRejectByReport, "Report Rejects as ClassB", url)
		postButtons += maybeButtonPOST(idspecsInReport, "Accept/Reject as ClassB", url)
	}

	// The only way to get embedded CGI args without them getting escaped is to submit a whole tag
	vizFormURL := "/fdb/visualize?"+rep.ToCGIArgs()
	vizFormTag := "<form action=\""+vizFormURL+"\" method=\"post\" target=\"_blank\">"

	jsonBytes,_ := json.MarshalIndent(rep.Options.GRS, "", "  ")
	rep.Debugf("--{ the GRS }--\n%s\n", string(jsonBytes))
	
	var params = map[string]interface{}{
		"R": rep,
		"Metadata": rep.MetadataTable(),
		"PostButtons": template.HTML(postButtons),
		"IdSpecs": template.HTML(strings.Join(idspecsAccepted,",")),
		"Title": "Reports (DB v2)",
		"UIOptions": opt,
		"VisualizationFormTag": template.HTML(vizFormTag),
	}
	if err := templates.ExecuteTemplate(w, "report-results", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}	
}

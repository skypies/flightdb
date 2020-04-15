package main

import(
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/context"

	"github.com/skypies/util/gcp/ds"
	hw "github.com/skypies/util/handlerware"

	"github.com/skypies/flightdb/config"
	"github.com/skypies/flightdb/ui"
)

var(
	GoogleCloudProjectId = "serfr0-fdb"
)

func init() {
	hw.RequireTls = false
	hw.InitTemplates("app/web/templates") // relative to go module root, which is git repo root

	// This is the routine that creates new contexts, and injects a provider into them,
	// as required by the FdbHandlers
	hw.CtxMakerCallback = func(r *http.Request) context.Context {
		ctx,_ := context.WithTimeout(r.Context(), 595 * time.Second)
		p,err := ds.NewCloudDSProvider(ctx, GoogleCloudProjectId)
		if err != nil {
			panic(fmt.Errorf("NewDB: could not get a clouddsprovider (projectId=%s): %v\n", GoogleCloudProjectId, err))
		}
		return ds.SetProvider(ctx, p)
	}

	// This stuff needs to be in sync with thew frontend app, which handles login
	hw.CookieName = "serfrfdb"
	hw.InitSessionStore(config.Get("sessions.key"), config.Get("sessions.prevkey"))
  hw.NoSessionHandler = loginRedirectHandler // redirects to frontend app, which has all the login config
  hw.InitGroup(hw.AdminGroup, config.Get("users.admin"))

	// ui/report - we host it here, to get batch server timeouts
	http.HandleFunc("/report",                    ui.WithFdbSession(ui.ReportHandler))

	// backend/batch.go
	http.HandleFunc("/batch/flights/dates",       ui.WithFdb(batchFlightDateRangeHandler))
	http.HandleFunc(batchDayUrl,                  ui.WithFdb(batchFlightDayHandler))
	http.HandleFunc(batchInstanceUrl,             ui.WithFdb(batchFlightHandler))

	// backend/bigquery.go (ran out of dispatch.yaml entries, so put this in 'batch')
	http.HandleFunc("/batch/publish-all-flights", ui.WithFdb(publishAllFlightsHandler))
	http.HandleFunc("/batch/publish-flights",     ui.WithFdb(publishFlightsHandler))

	// backend/foia.go
	//http.HandleFunc("/foia/load",                 ui.WithFdb(foiaHandler))
	//http.HandleFunc("/foia/enqueue",              ui.WithFdb(multiEnqueueHandler))
	//http.HandleFunc("/foia/rm",                   ui.WithFdb(rmHandler))


	// STOLEN from app/frontend, just so we can run locally for testing; app/dispatch.yaml
	// will prevent any of these URLs coming to this app when it is deployed to appengine

	// ui/api.go
	http.HandleFunc("/fdb/vector",          ui.WithFdb(ui.VectorHandler))
	http.HandleFunc("/api/flight/lookup",   ui.WithFdb(ui.FlightLookupHandler))
	http.HandleFunc("/api/procedures",      ui.WithFdb(ui.ProcedureHandler))

	// ui/tracks.go
	http.HandleFunc("/fdb/tracks",          ui.WithFdb(ui.TrackHandler))
	http.HandleFunc("/fdb/trackset",        ui.WithFdb(ui.TracksetHandler))

	// ui/map.go
	http.HandleFunc("/fdb/map",             hw.WithCtx(ui.MapHandler))

	// ui/debug.go
	http.HandleFunc("/fdb/debug",           ui.WithFdb(ui.DebugHandler))  // fdb/text ??
	http.HandleFunc("/fdb/sched",           ui.WithFdb(ui.DebugSchedHandler))  // fdb/text ??

	// ui/georestrictorsets.go
	stem := "/fdb/restrictors"
	http.HandleFunc(stem+"/list",           ui.WithFdbSession(ui.RListHandler))
	http.HandleFunc(stem+"/grs/new",        ui.WithFdbSession(ui.RGrsNewHandler))
	http.HandleFunc(stem+"/grs/delete",     ui.WithFdbSession(ui.RGrsDeleteHandler))
	http.HandleFunc(stem+"/grs/edit",       ui.WithFdbSession(ui.RGrsEditHandler))
	http.HandleFunc(stem+"/grs/view",       ui.WithFdbSession(ui.RGrsViewHandler))
	http.HandleFunc(stem+"/gr/new",         ui.WithFdbSession(ui.RGrNewHandler))
	http.HandleFunc(stem+"/gr/edit",        ui.WithFdbSession(ui.RGrEditHandler))
	http.HandleFunc(stem+"/gr/delete",      ui.WithFdbSession(ui.RGrDeleteHandler))

	// ui/historical.go
	http.HandleFunc("/fdb/historical",      ui.WithFdb(ui.HistoricalHandler))

	// ui/json.go
	http.HandleFunc("/fdb/json",            ui.WithFdb(ui.JsonHandler))
	http.HandleFunc("/fdb/snarf",           ui.WithFdb(ui.SnarfHandler))

	// ui/lists.go
	http.HandleFunc("/fdb/list",            ui.WithFdb(ui.ListHandler))

	// ui/sideview.go
	http.HandleFunc("/fdb/sideview",        ui.WithFdb(ui.SideviewHandler))

	// ui/visualize.go
	http.HandleFunc("/fdb/visualize",       ui.WithFdb(ui.VisualizeHandler))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fs := http.FileServer(http.Dir("./app/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Printf("Listening on port %s [flightdb/app/backend]", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func loginRedirectHandler (ctx context.Context, w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/fdb/login", http.StatusFound)
}

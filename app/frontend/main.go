package main

import(
	"fmt"
	"html/template"
	"net/http"
	"os"
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/skypies/util/gcp/ds"
	hw "github.com/skypies/util/handlerware"

	_ "github.com/skypies/flightdb/analysis" // populate the reports registry
	"github.com/skypies/flightdb/ui"
	"github.com/skypies/pi/airspace/realtime"
)

var(
	tmpl *template.Template  // Singleton that belongs to the webapp
	GoogleCloudProjectId = "serfr0-fdb"
)

// FIXME: This is a one-off thing, for airspace/realtime; should kill it off
type hackTemplateHandler func(http.ResponseWriter, *http.Request, *template.Template)
func hackHandleWithTemplates(tmpl *template.Template, th hackTemplateHandler) hw.BaseHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		th(w,r,tmpl)
	}
}
	
func init() {
	hw.InitTemplates("app/frontend/templates") // location relative to go module root, which is git repo root
	tmpl = hw.Templates

	// This is the routine that creates new contexts, and injects a provider into them,
	// as required by the FdbHandlers
	//ctxMakerAE := func(r *http.Request) context.Context {
	//	ctx := appengineds.CtxMakerFunc(r)
	//	return ds.SetProvider(ctx, appengineds.AppengineDSProvider{}) 
	//}
	hw.CtxMakerCallback = func(r *http.Request) context.Context {
		ctx,_ := context.WithTimeout(r.Context(), 55 * time.Second)
		p,err := ds.NewCloudDSProvider(ctx, GoogleCloudProjectId)
		if err != nil {
			panic(fmt.Errorf("NewDB: could not get a clouddsprovider (projectId=%s): %v\n", GoogleCloudProjectId, err))
		}
		return ds.SetProvider(ctx, p) 
	}
	
	http.HandleFunc("/", hackHandleWithTemplates(tmpl, realtime.AirspaceHandler))

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
	
	// fr24poller.go
	http.HandleFunc("/api/fr24",            ui.WithFdbSession(fr24PollHandler)) // FIXME: AdminAccess
	http.HandleFunc("/api/fr24q",           ui.WithFdbSession(fr24QueryHandler))
	http.HandleFunc("/api/schedcache/view", ui.WithFdb(schedcacheViewHandler))

	// metar.go
	http.HandleFunc("/api/metar/lookup",    ui.WithFdb(metarLookupHandler))
	http.HandleFunc("/api/metar/lookupall", ui.WithFdb(metarLookupAllHandler))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fs := http.FileServer(http.Dir("./app/frontend/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Printf("Listening on port %s [flightdb/app/frontend]", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

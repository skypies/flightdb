package main

import(
	"fmt"
	"net/http"
	"os"
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/skypies/util/gcp/ds"
	hw "github.com/skypies/util/handlerware"
	"github.com/skypies/util/login"

	_ "github.com/skypies/flightdb/analysis" // populate the reports registry
	"github.com/skypies/flightdb/config"
	"github.com/skypies/flightdb/ui"
	"github.com/skypies/pi/airspace/realtime"
)

var(
	GoogleCloudProjectId = "serfr0-fdb"
)

func init() {
	hw.RequireTls = false
	hw.InitTemplates("app/frontend/templates") // location relative to go module root, which is git repo root

	// The FdbHandlers expect to find a DSProvider in the context
	hw.CtxMakerCallback = func(r *http.Request) context.Context {
		ctx,_ := context.WithTimeout(r.Context(), 55 * time.Second)
		p,err := ds.NewCloudDSProvider(ctx, GoogleCloudProjectId)
		if err != nil {
			panic(fmt.Errorf("NewDB: could not get a clouddsprovider (projectId=%s): %v\n", GoogleCloudProjectId, err))
		}
		return ds.SetProvider(ctx, p)
	}


	hw.CookieName = "serfrfdb"
  hw.NoSessionHandler = loginPageHandler
	hw.InitSessionStore(config.Get("sessions.key"), config.Get("sessions.prevkey"))
  hw.InitGroup(hw.AdminGroup, config.Get("users.admin"))

	login.OnSuccessCallback = func(w http.ResponseWriter, r *http.Request, email string) error {
		hw.CreateSession(r.Context(), w, r, hw.UserSession{Email:email})
		return nil
	}
	login.Host                  = "http://fdb.serfr1.org"
	// login.Host                  = "http://localhost:8080"
	login.RedirectUrlStem       = "/fdb/login" // oauth2 callbacks will register  under here
	login.AfterLoginRelativeUrl = "/fdb/home" // where the user finally ends up, after being logged in
	login.GoogleClientID        = config.Get("google.oauth2.appid")
	login.GoogleClientSecret    = config.Get("google.oauth2.secret")
	login.Init()

	http.HandleFunc("/fdb/login",           hw.WithCtx(loginPageHandler))
	http.HandleFunc("/fdb/logout",          hw.WithCtx(logoutHandler))

	http.HandleFunc("/fdb/home",            hw.WithCtx(flatPageHandler))
	http.HandleFunc("/fdb/privacy",         hw.WithCtx(flatPageHandler))
	http.HandleFunc("/fdb/debug2",          hw.WithCtx(DebugSessionHandler))
	http.HandleFunc("/fdb/debug3",          ui.WithFdbSession(DebugFdbSessionHandler))

	// This handler comes from pi/
	http.HandleFunc("/",                    hw.WithCtx(realtime.AirspaceHandler))

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

// To add a new flat page to a URL that is routed to this app:
//  1. add http.HandleFunc(URL, flatPageHandler)
//  2. Add a new template, and have it match the URL, e.g. {{define "/f/foobar"}}
func flatPageHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	templates := hw.GetTemplates(ctx)
	pagename := r.URL.Path
	if err := templates.ExecuteTemplate(w, pagename, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func loginPageHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {	
	templates := hw.GetTemplates(ctx)
	var params = map[string]interface{}{
		"google": login.Goauth2.GetLoginUrl(w,r),
		"googlefromscratch": login.Goauth2.GetLogoutUrl(w,r),
	}

	if err := templates.ExecuteTemplate(w, "login", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func logoutHandler (ctx context.Context, w http.ResponseWriter, r *http.Request) {
	hw.OverwriteSessionToNil(ctx, w, r)
	http.Redirect(w, r, "/fdb/home", http.StatusFound)
}

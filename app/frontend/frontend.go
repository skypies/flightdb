package frontend

import(
	"html/template"
	"net/http"

	_ "github.com/skypies/flightdb/analysis" // populate the reports registry
	"github.com/skypies/flightdb/ui"
	mytemplates "github.com/skypies/flightdb/templates"         // parse up the templates
	"github.com/skypies/pi/airspace/realtime"
)

var AppTemplates *template.Template  // Singleton that belongs to the webapp

// This is a one-off thing, for airspace/realtime; should kill it off
type templateHandler func(http.ResponseWriter, *http.Request, *template.Template)
type baseHandler     func(http.ResponseWriter, *http.Request)
func handleWithTemplates(tmpl *template.Template, th templateHandler) baseHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		th(w,r,tmpl)
	}
}

func init() {
	AppTemplates = mytemplates.LoadTemplates("templates")
	
	http.HandleFunc("/", handleWithTemplates(AppTemplates, realtime.AirspaceHandler))

	// ui/api.go
	http.HandleFunc("/fdb/vector", ui.WithCtxOpt(ui.VectorHandler))
	http.HandleFunc("/api/flight/lookup", ui.WithCtxOpt(ui.FlightLookupHandler))
	http.HandleFunc("/api/procedures", ui.WithCtxOpt(ui.ProcedureHandler))

	// ui/tracks.go
	http.HandleFunc("/fdb/tracks", ui.WithCtxOptTmpl(AppTemplates, ui.TrackHandler))
	http.HandleFunc("/fdb/trackset", ui.WithCtxOptTmpl(AppTemplates, ui.TracksetHandler))

	// ui/map.go
	http.HandleFunc("/fdb/map", ui.WithCtxTmpl(AppTemplates, ui.MapHandler))

	// ui/debug.go
	http.HandleFunc("/fdb/debug", ui.WithCtxOpt(ui.DebugHandler))  // fdb/text ??

	// ui/georestrictorsets.go
	stem := "/fdb/restrictors"
	http.HandleFunc(stem+"/list", ui.WithCtxOptTmplUser(AppTemplates, ui.RListHandler))
	http.HandleFunc(stem+"/grs/new", ui.WithCtxOptTmplUser(AppTemplates, ui.RGrsNewHandler))
	http.HandleFunc(stem+"/grs/delete",ui.WithCtxOptTmplUser(AppTemplates, ui.RGrsDeleteHandler))
	http.HandleFunc(stem+"/grs/edit", ui.WithCtxOptTmplUser(AppTemplates, ui.RGrsEditHandler))
	http.HandleFunc(stem+"/grs/view", ui.WithCtxOptTmplUser(AppTemplates, ui.RGrsViewHandler))
	http.HandleFunc(stem+"/gr/new", ui.WithCtxOptTmplUser(AppTemplates, ui.RGrNewHandler))
	http.HandleFunc(stem+"/gr/edit", ui.WithCtxOptTmplUser(AppTemplates, ui.RGrEditHandler))
	http.HandleFunc(stem+"/gr/delete", ui.WithCtxOptTmplUser(AppTemplates, ui.RGrDeleteHandler))

	// ui/historical.go
	http.HandleFunc("/fdb/historical", ui.WithCtxTmpl(AppTemplates, ui.HistoricalHandler))

	// ui/json.go
	http.HandleFunc("/fdb/json", ui.WithCtxOpt(ui.JsonHandler))
	http.HandleFunc("/fdb/snarf", ui.WithCtxOpt(ui.SnarfHandler))

	// ui/lists.go
	http.HandleFunc("/fdb/list", ui.WithCtxTmpl(AppTemplates, ui.ListHandler))

	// ui/sideview.go
	http.HandleFunc("/fdb/sideview",  ui.WithCtxOpt(ui.SideviewHandler))

	// ui/visualize.go
	http.HandleFunc("/fdb/visualize", ui.WithCtxOpt(ui.VisualizeHandler))

}

// TODO: rename TracksetHandler and VectorHandler (and perhaps /fdb/tracks[et])
// TODO: eliminate ui.HandlwWithTemplates

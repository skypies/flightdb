package frontend

import(
	"html/template"
	"net/http"

	"golang.org/x/net/context"
	
	"github.com/skypies/util/ae"
	appengineds "github.com/skypies/util/ae/ds"
	"github.com/skypies/util/gcp/ds"
	"github.com/skypies/util/widget"

	_ "github.com/skypies/flightdb/analysis" // populate the reports registry
	"github.com/skypies/flightdb/ui"
	"github.com/skypies/pi/airspace/realtime"
)

var(
	tmpl *template.Template  // Singleton that belongs to the webapp
)

// This is a one-off thing, for airspace/realtime; should kill it off
type hackTemplateHandler func(http.ResponseWriter, *http.Request, *template.Template)
func hackHandleWithTemplates(tmpl *template.Template, th hackTemplateHandler) widget.BaseHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		th(w,r,tmpl)
	}
}
	
func init() {
	// Templates are kinda messy.
	// The functions to parse them live in the UI library.
	// The "templates" dir lives under the appengine app's main dir; to reuse templates
	// from other places, we symlink them underneath this.
	tmpl = widget.ParseRecursive(template.New("").Funcs(ui.TemplateFuncMap()), "templates")

	// This is the routine that creates new contexts, and injects a provider into them,
	// as required by the FdbHandlers
	ctxMaker := func(r *http.Request) context.Context {
		ctx := appengineds.CtxMakerFunc(r)
		return ds.SetProvider(ctx, appengineds.AppengineDSProvider{}) 
	}

	http.HandleFunc("/", hackHandleWithTemplates(tmpl, realtime.AirspaceHandler))

	// ui/api.go
	http.HandleFunc("/fdb/vector", ui.WithFdbCtxOpt(ctxMaker, ui.VectorHandler))
	http.HandleFunc("/api/flight/lookup", ui.WithFdbCtxOpt(ctxMaker, ui.FlightLookupHandler))
	http.HandleFunc("/api/procedures", ui.WithFdbCtxOpt(ctxMaker, ui.ProcedureHandler))

	// ui/tracks.go
	http.HandleFunc("/fdb/tracks", ui.WithFdbCtxOptTmpl(ctxMaker, tmpl, ui.TrackHandler))
	http.HandleFunc("/fdb/trackset", ui.WithFdbCtxOptTmpl(ctxMaker, tmpl, ui.TracksetHandler))

	// ui/map.go
	http.HandleFunc("/fdb/map", widget.WithCtxTmpl(ctxMaker, tmpl, ui.MapHandler))

	// ui/debug.go
	http.HandleFunc("/fdb/debug", ui.WithFdbCtxOpt(ctxMaker, ui.DebugHandler))  // fdb/text ??
	http.HandleFunc("/fdb/sched", ui.WithFdbCtxOpt(ctxMaker, ui.DebugSchedHandler))  // fdb/text ??
	http.HandleFunc("/fdb/sched2", ui.WithFdbCtxOpt(ctxMaker, ui.DebugNewSchedHandler))  // fdb/text ??

	// ui/georestrictorsets.go
	stem := "/fdb/restrictors"
	http.HandleFunc(stem+"/list", ui.WithFdbCtxOptTmplUser(ctxMaker, tmpl, ui.RListHandler))
	http.HandleFunc(stem+"/grs/new", ui.WithFdbCtxOptTmplUser(ctxMaker, tmpl, ui.RGrsNewHandler))
	http.HandleFunc(stem+"/grs/delete",ui.WithFdbCtxOptTmplUser(ctxMaker, tmpl, ui.RGrsDeleteHandler))
	http.HandleFunc(stem+"/grs/edit", ui.WithFdbCtxOptTmplUser(ctxMaker, tmpl, ui.RGrsEditHandler))
	http.HandleFunc(stem+"/grs/view", ui.WithFdbCtxOptTmplUser(ctxMaker, tmpl, ui.RGrsViewHandler))
	http.HandleFunc(stem+"/gr/new", ui.WithFdbCtxOptTmplUser(ctxMaker, tmpl, ui.RGrNewHandler))
	http.HandleFunc(stem+"/gr/edit", ui.WithFdbCtxOptTmplUser(ctxMaker, tmpl, ui.RGrEditHandler))
	http.HandleFunc(stem+"/gr/delete", ui.WithFdbCtxOptTmplUser(ctxMaker, tmpl, ui.RGrDeleteHandler))

	// ui/historical.go
	http.HandleFunc("/fdb/historical", ui.WithFdbCtxTmpl(ctxMaker, tmpl, ui.HistoricalHandler))

	// ui/json.go
	http.HandleFunc("/fdb/json", ui.WithFdbCtxOpt(ctxMaker, ui.JsonHandler))
	http.HandleFunc("/fdb/snarf", ui.WithFdbCtxOpt(ctxMaker, ui.SnarfHandler))

	// ui/lists.go
	http.HandleFunc("/fdb/list", ui.WithFdbCtxTmpl(ctxMaker, tmpl, ui.ListHandler))

	// ui/sideview.go
	http.HandleFunc("/fdb/sideview",  ui.WithFdbCtxOpt(ctxMaker, ui.SideviewHandler))

	// ui/visualize.go
	http.HandleFunc("/fdb/visualize", ui.WithFdbCtxOptTmpl(ctxMaker, tmpl, ui.VisualizeHandler))

	http.HandleFunc("/fdb/memcachesingleton", ui.WithCtxOpt(ctxMaker, ae.SaveSingletonToMemcacheHandler))
}

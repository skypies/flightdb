package backend

import(
	"html/template"
	"net/http"

	"golang.org/x/net/context"

	appengineds "github.com/skypies/util/ae/ds"
	"github.com/skypies/util/gcp/ds"
	"github.com/skypies/util/widget"

	"github.com/skypies/flightdb/ui"
)

var tmpl *template.Template // Could this be a local inside init() ?

func init() {
	tmpl = widget.ParseRecursive(template.New("").Funcs(ui.TemplateFuncMap()), "templates")

	// This is the routine that creates new contexts, and injects a provider into them,
	// as required by the FdbHandlers
	ctxMaker := func(r *http.Request) context.Context {
		ctx := appengineds.CtxMakerFunc(r)
		return ds.SetProvider(ctx, appengineds.AppengineDSProvider{}) 
	}

	// ui/report - we host it here, to get batch server timeouts
	http.HandleFunc("/report", ui.WithFdbCtxOptTmpl(ctxMaker, tmpl, ui.ReportHandler))

	// backend/batch.go
	http.HandleFunc("/batch/flights/dates",  ui.WithFdbCtx(ctxMaker, batchFlightDateRangeHandler))
	http.HandleFunc(batchDayUrl,             ui.WithFdbCtx(ctxMaker, batchFlightDayHandler))
	http.HandleFunc(batchInstanceUrl,        ui.WithFdbCtx(ctxMaker, batchFlightHandler))

	// backend/bigquery.go (ran out of dispatch.yaml entries, so put this in 'batch')
	http.HandleFunc("/batch/publish-all-flights", ui.WithFdbCtx(ctxMaker, publishAllFlightsHandler))
	http.HandleFunc("/batch/publish-flights",     ui.WithFdbCtx(ctxMaker, publishFlightsHandler))

	// backend/foia.go
	http.HandleFunc("/foia/load", ui.WithFdbCtx(ctxMaker, foiaHandler))
	http.HandleFunc("/foia/enqueue", ui.WithFdbCtx(ctxMaker, multiEnqueueHandler))
	//http.HandleFunc("/foia/rm", ui.WithFdbCtx(ctxMaker, rmHandler))
	
	// backend/fr24poller.go
	http.HandleFunc("/be/fr24", ui.WithFdbCtx(ctxMaker, fr24PollHandler))
	http.HandleFunc("/be/fr24q", ui.WithFdbCtx(ctxMaker, fr24QueryHandler))
	http.HandleFunc("/be/schedcache/view", ui.WithFdbCtx(ctxMaker, schedcacheViewHandler))

	// backend/metar.go
	http.HandleFunc("/metar/lookup", ui.WithFdbCtx(ctxMaker, metarLookupHandler))
	http.HandleFunc("/metar/lookupall", ui.WithFdbCtx(ctxMaker, metarLookupAllHandler))
}

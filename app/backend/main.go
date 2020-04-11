package main

import(
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/context"

	"github.com/skypies/util/gcp/ds"
	hw "github.com/skypies/util/handlerware"

	"github.com/skypies/flightdb/ui"
)

var(
	tmpl *template.Template // Could this be a local inside init() ?
	GoogleCloudProjectId = "serfr0-fdb"
)

func init() {
	// For go111, appengine uses the module root, which is the root of the git repo; so
	// the relative dirname for templates is relative to the root of the git repo.
	tmpl = hw.ParseRecursive(template.New("").Funcs(ui.TemplateFuncMap()), "app/backend/templates")

	// This is the routine that creates new contexts, and injects a provider into them,
	// as required by the FdbHandlers
	//ctxMaker := func(r *http.Request) context.Context {
	//	ctx := appengineds.CtxMakerFunc(r)
	//	return ds.SetProvider(ctx, appengineds.AppengineDSProvider{}) 
	//}
	ctxMaker := func(r *http.Request) context.Context {
		ctx,_ := context.WithTimeout(r.Context(), 595 * time.Second)
		p,err := ds.NewCloudDSProvider(ctx, GoogleCloudProjectId)
		if err != nil {
			panic(fmt.Errorf("NewDB: could not get a clouddsprovider (projectId=%s): %v\n", GoogleCloudProjectId, err))
		}
		return ds.SetProvider(ctx, p) 
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
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fs := http.FileServer(http.Dir("./app/backend/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Printf("Listening on port %s [flightdb/app/backend]", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

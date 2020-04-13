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

	"github.com/skypies/flightdb/ui"
)

var(
	GoogleCloudProjectId = "serfr0-fdb"
)

func init() {
	hw.InitTemplates("app/backend/templates") // relative to go module root, which is git repo root

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

	// ui/report - we host it here, to get batch server timeouts
	http.HandleFunc("/report",                    ui.WithFdb(ui.ReportHandler))

	// backend/batch.go
	http.HandleFunc("/batch/flights/dates",       ui.WithFdb(batchFlightDateRangeHandler))
	http.HandleFunc(batchDayUrl,                  ui.WithFdb(batchFlightDayHandler))
	http.HandleFunc(batchInstanceUrl,             ui.WithFdb(batchFlightHandler))

	// backend/bigquery.go (ran out of dispatch.yaml entries, so put this in 'batch')
	http.HandleFunc("/batch/publish-all-flights", ui.WithFdb(publishAllFlightsHandler))
	http.HandleFunc("/batch/publish-flights",     ui.WithFdb(publishFlightsHandler))

	// backend/foia.go
	http.HandleFunc("/foia/load",                 ui.WithFdb(foiaHandler))
	http.HandleFunc("/foia/enqueue",              ui.WithFdb(multiEnqueueHandler))
	//http.HandleFunc("/foia/rm",                   ui.WithFdb(rmHandler))
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

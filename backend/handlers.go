package main

import(
	"net/http"

	"github.com/skypies/flightdb2/fgae"
)

func init() {
	http.HandleFunc("/fdb/batch/run", fgae.BatchHandler)
	http.HandleFunc("/fdb/batch/instance", fgae.BatchInstanceHandler)
}

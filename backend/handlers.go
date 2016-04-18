package main

import(
	"net/http"

	"github.com/skypies/flightdb2/fgae"
)

func init() {
	// OLD
	http.HandleFunc("/fdb/batch/run", fgae.BatchHandler)
	http.HandleFunc("/fdb/batch/instance", fgae.BatchInstanceHandler)

	// NEW
	http.HandleFunc(fgae.RangeUrl,    fgae.BatchFlightDateRangeHandler)
	http.HandleFunc(fgae.DayUrl,      fgae.BatchFlightDayHandler)
	http.HandleFunc(fgae.InstanceUrl, fgae.BatchFlightHandler)
}

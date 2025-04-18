package main

import(
	"fmt"
	"net/http"

	"context"

	hw "github.com/skypies/util/handlerware"

	"github.com/skypies/flightdb/fgae"
)

func DebugFdbSessionHandler(db fgae.FlightDB, w http.ResponseWriter, r *http.Request) {
	DebugSessionHandler(db.Ctx(), w, r)
}

func DebugSessionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	sesh,seshOK := hw.GetUserSession(ctx)

	str := fmt.Sprintf("OK!\n\nseshOK: %v\nsesh: %#v\n--\n", seshOK, sesh)
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(str))
}

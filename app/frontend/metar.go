package frontend

import(
	"fmt"
	"net/http"
	"time"

	"github.com/skypies/util/date"
	"github.com/skypies/util/widget"

	"github.com/skypies/flightdb/fgae"
	"github.com/skypies/flightdb/metar"
)

// {{{ metarLookupHandler

// /metar/lookup [?t=123123123123] [&loc=KSFO]
//  [&h=3]  offset hour (defaults to now)
//  [&n=6]  number of hours to lookup in an archive
func metarLookupHandler(db fgae.FlightDB, w http.ResponseWriter, r *http.Request) {
	ctx := db.Ctx()
	p := db.Backend
	str := "OK\n--\n\n"

	loc := r.FormValue("loc")
	if loc == "" { loc = metar.DefaultStation }

	t := time.Now().UTC()
	if hours := widget.FormValueInt64(r,"h"); hours > 0 {
		t = t.Add(time.Duration(-1 * hours) * time.Hour)
	}
	if r.FormValue("t") != "" {
		t = widget.FormValueEpochTime(r, "t")
	}

	str += fmt.Sprintf("Lookup for loc=%s, t=%s (%s)\n", loc, t, date.InPdt(t))

	mr,err,deb := metar.LookupOrFetch(ctx, p, loc, t)

	str += fmt.Sprintf("LookupOrFetch Result: %s\nLookupOrFetch Err: %v\n\n%s", mr, err, deb)

	mr2,err2 := metar.LookupReport(ctx, p, loc, t)
	str += fmt.Sprintf("\n********\n\nLookupReport Result: %s\nLookup Err: %v\n\n", mr2, err2)

	db.Infof(str)

	if n := widget.FormValueInt64(r, "n"); n > 0 {
		s, e := t.Add(time.Duration(-1 * n) * time.Hour), t

		ar,err := metar.LookupArchive(ctx, p, loc, s, e)
		str += fmt.Sprintf("\n********\n\nLookupArchive Result: %s\nLookup Err: %v\n\n", ar, err)		
	}
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(str))
}

// }}}
// {{{ metarLookupAllHandler

// /metar/lookupall?
//    n=17                (num days)
//  [&t=123981723129837]
//  [&loc=KSFO]
func metarLookupAllHandler(db fgae.FlightDB, w http.ResponseWriter, r *http.Request) {
	ctx := db.Ctx()
	p := db.Backend
	str := "OK\n--\n\n"

	n := widget.FormValueInt64(r, "n");
	if n == 0 { n = 1 }

	loc := r.FormValue("loc")
	if loc == "" { loc = metar.DefaultStation }

	t := time.Now().UTC()
	if r.FormValue("t") != "" {
		t = widget.FormValueEpochTime(r, "t")
	}
	
	str += fmt.Sprintf("LookupAll for loc=%s, t=%s (%s)\n\n", loc, t, date.InPdt(t))

	for ; n>0; n-- {
		dr,err := metar.LookupDayReport(ctx, p, loc, t)
		str += fmt.Sprintf("%s [err:%v]\n", dr, err)
		t = t.AddDate(0,0,-1)
	}
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(str))
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
